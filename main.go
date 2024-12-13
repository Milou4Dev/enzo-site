package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
}

type App struct {
	config      Config
	router      *gin.Engine
	rateLimiter *RateLimiter
	logger      *Logger
}

type RateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
}

type Logger struct {
	*log.Logger
}

func NewApp() *App {
	config := loadConfig()
	gin.SetMode(gin.ReleaseMode)

	app := &App{
		config:      config,
		router:      gin.New(),
		rateLimiter: newRateLimiter(config.RateLimit, config.RateLimitBurst),
		logger:      newLogger(),
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
		TrustedProxies:   []string{"127.0.0.1"},
		MaxRequestSize:   1 << 20,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     10 * time.Second,
		IdleTimeout:      120 * time.Second,
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
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (app *App) setupMiddleware() {
	app.router.Use(gin.Recovery())
	app.router.Use(app.requestLogger())
	app.router.Use(app.securityHeaders())
	app.router.Use(app.rateLimiterMiddleware())
	app.router.Use(app.cacheControl())
	app.router.Use(app.requestSizeLimit())
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
		Addr:         ":" + app.config.Port,
		Handler:      app.router,
		ReadTimeout:  app.config.ReadTimeout,
		WriteTimeout: app.config.WriteTimeout,
		IdleTimeout:  app.config.IdleTimeout,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		<-sigChan
		ctx, cancel := context.WithTimeout(context.Background(), app.config.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			app.logger.Printf("Server shutdown error: %v", err)
		}
	}()

	app.logger.Printf("Server starting on port %s", app.config.Port)
	return server.ListenAndServe()
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
