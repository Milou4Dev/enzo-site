package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

type Config struct {
	Port             string
	RateLimit        int
	RateLimitBurst   int
	StaticDir        string
	TemplatesPattern string
	ShutdownTimeout  time.Duration
	TrustedProxies   []string
	MaxRequestSize   int64
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	MaxConns         int
	CircuitTimeout   time.Duration
	CircuitMaxFails  int
}

type App struct {
	config      Config
	router      *gin.Engine
	rateLimiter *RateLimiter
	logger      *Logger
	metrics     *Metrics
	circuit     *CircuitBreaker
	pool        sync.Pool
}

type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

type Logger struct {
	*log.Logger
	errorChan chan error
}

type Metrics struct {
	requestCount   uint64
	errorCount     uint64
	responseTime   time.Duration
	activeRequests int64
	mu             sync.RWMutex
}

type CircuitBreaker struct {
	failures int32
	lastFail time.Time
	timeout  time.Duration
	maxFails int32
	mu       sync.RWMutex
}

func NewApp() *App {
	config := loadConfig()
	gin.SetMode(gin.ReleaseMode)

	app := &App{
		config:      config,
		router:      gin.New(),
		rateLimiter: newRateLimiter(config.RateLimit, config.RateLimitBurst),
		logger:      newLogger(),
		metrics:     newMetrics(),
		circuit:     newCircuitBreaker(config.CircuitTimeout, config.CircuitMaxFails),
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
	}

	app.setupMiddleware()
	app.setupRoutes()

	return app
}

func loadConfig() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		RateLimit:        5,
		RateLimitBurst:   10,
		StaticDir:        "static",
		TemplatesPattern: "templates/*",
		ShutdownTimeout:  10 * time.Second,
		TrustedProxies:   strings.Split(getEnv("TRUSTED_PROXIES", "127.0.0.1"), ","),
		MaxRequestSize:   1 << 20,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     10 * time.Second,
		IdleTimeout:      120 * time.Second,
		MaxConns:         1000,
		CircuitTimeout:   30 * time.Second,
		CircuitMaxFails:  5,
	}
}

func newRateLimiter(limit, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate.Limit(limit),
		burst: burst,
	}
}

func newLogger() *Logger {
	return &Logger{
		Logger:    log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.LUTC),
		errorChan: make(chan error, 100),
	}
}

func newMetrics() *Metrics {
	return &Metrics{}
}

func newCircuitBreaker(timeout time.Duration, maxFails int) *CircuitBreaker {
	return &CircuitBreaker{
		timeout:  timeout,
		maxFails: int32(maxFails),
	}
}

func (app *App) setupMiddleware() {
	app.router.Use(gin.Recovery())
	app.router.Use(app.metricsMiddleware())
	app.router.Use(app.requestLogger())
	app.router.Use(app.securityHeaders())
	app.router.Use(app.rateLimiterMiddleware())
	app.router.Use(app.circuitBreakerMiddleware())
	app.router.Use(app.cacheControl())
	app.router.Use(app.requestSizeLimit())
	app.router.Use(app.corsMiddleware())
}

func (app *App) setupRoutes() {
	app.router.LoadHTMLGlob(app.config.TemplatesPattern)
	app.router.StaticFS("/static", gin.Dir(app.config.StaticDir, false))
	app.router.GET("/", app.handleHome)
	app.router.GET("/health", app.handleHealth)
	app.router.NoRoute(app.handle404)
}

func (app *App) Run() error {
	server := &http.Server{
		Addr:           ":" + app.config.Port,
		Handler:        app.router,
		ReadTimeout:    app.config.ReadTimeout,
		WriteTimeout:   app.config.WriteTimeout,
		IdleTimeout:    app.config.IdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				app.metrics.mu.RLock()
				app.logger.Printf("Metrics - Requests: %d, Errors: %d, Active: %d",
					atomic.LoadUint64(&app.metrics.requestCount),
					atomic.LoadUint64(&app.metrics.errorCount),
					atomic.LoadInt64(&app.metrics.activeRequests))
				app.metrics.mu.RUnlock()
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-app.logger.errorChan:
				app.logger.Printf("Error: %v", err)
			}
		}
	})

	g.Go(func() error {
		app.logger.Printf("Server starting on port %s", app.config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %v", err)
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), app.config.ShutdownTimeout)
		defer cancel()

		app.logger.Print("Shutting down server...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %v", err)
		}
		return nil
	})

	return g.Wait()
}

func (app *App) handleHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func (app *App) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (app *App) handle404(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", nil)
}

func (app *App) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		app.logger.Printf("[%s] %s %s %d %v",
			c.Request.Method,
			path,
			c.ClientIP(),
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	limiter, _ := rl.limiters.LoadOrStore(key, rate.NewLimiter(rl.rate, rl.burst))
	return limiter.(*rate.Limiter).Allow()
}

func (app *App) rateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !app.rateLimiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"retry_after": "1s",
			})
			return
		}
		c.Next()
	}
}

func (app *App) cacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=31536000")
		} else {
			c.Header("Cache-Control", "no-store, must-revalidate")
		}
		c.Next()
	}
}

func (app *App) requestSizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > app.config.MaxRequestSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request size exceeds maximum allowed size of %d bytes", app.config.MaxRequestSize),
			})
			return
		}
		c.Next()
	}
}

func (app *App) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; script-src 'self' https://cdnjs.cloudflare.com; connect-src 'self' https://formsubmit.co; img-src 'self' data: https:; font-src 'self' https://cdnjs.cloudflare.com; frame-src 'none'; object-src 'none'; base-uri 'self'; form-action 'self' https://formsubmit.co;")
		c.Next()
	}
}

func (app *App) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		atomic.AddInt64(&app.metrics.activeRequests, 1)
		defer atomic.AddInt64(&app.metrics.activeRequests, -1)

		c.Next()

		atomic.AddUint64(&app.metrics.requestCount, 1)
		if c.Writer.Status() >= 400 {
			atomic.AddUint64(&app.metrics.errorCount, 1)
		}

		app.metrics.mu.Lock()
		app.metrics.responseTime += time.Since(start)
		app.metrics.mu.Unlock()

		if c.Writer.Status() >= 500 {
			app.circuit.recordFailure()
		}
	}
}

func (app *App) circuitBreakerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !app.circuit.allow() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service temporarily unavailable",
			})
			return
		}
		c.Next()
	}
}

func (app *App) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	failures := atomic.LoadInt32(&cb.failures)
	if failures >= cb.maxFails {
		if time.Since(cb.lastFail) > cb.timeout {
			atomic.StoreInt32(&cb.failures, 0)
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt32(&cb.failures, 1)
	cb.lastFail = time.Now()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	app := NewApp()
	if err := app.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
