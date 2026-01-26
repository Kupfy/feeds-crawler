package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware creates a logging middleware
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Get request path
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get method
		method := c.Request.Method

		// Build query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Get user ID if authenticated
		userID := ""
		if uid, exists := c.Get("userID"); exists {
			userID = uid.(string)
		}

		// Log format: [timestamp] status method path latency clientIP userID
		log.Printf("[%s] %d %s %s %v %s user=%s",
			startTime.Format(time.RFC3339),
			statusCode,
			method,
			path,
			latency,
			clientIP,
			userID,
		)

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Printf("[ERROR] %s", err.Error())
			}
		}
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				c.JSON(500, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
