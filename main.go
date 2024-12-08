package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())
	r.Use(securityHeaders())
	r.Use(rateLimiter())
	r.Use(cacheControl())
	r.LoadHTMLGlob("templates/*")
	r.StaticFS("/static", gin.Dir("static", false))
	r.GET("/", handleHome)
	r.GET("/health", handleHealth)
	r.NoRoute(handle404)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Printf("Server failed to start: %v", err)
		os.Exit(1)
	}
}

func handleHome(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{
		"title": "Site Web",
	})
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func handle404(c *gin.Context) {
	c.HTML(404, "404.html", nil)
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		log.Printf("[%s] %s %s %d %v",
			c.Request.Method,
			path,
			c.ClientIP(),
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

func rateLimiter() gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Second), 5)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatus(429)
			return
		}
		c.Next()
	}
}

func cacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=31536000")
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; "+
				"script-src 'self' https://cdnjs.cloudflare.com; "+
				"connect-src 'self' https://formsubmit.co; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' https://cdnjs.cloudflare.com; "+
				"frame-src 'none'; "+
				"object-src 'none';")
		c.Next()
	}
}
