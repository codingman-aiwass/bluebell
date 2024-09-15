package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

var userRequestTimes = make(map[string]time.Time)
var mu sync.Mutex

//func BlockingRateLimitMiddleware(rl ratelimit.Limiter) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// Wait for the limiter to allow the request
//		rl.Take()
//
//		// Continue to the next handler if allowed
//		c.Next()
//	}
//}

func NonBlockingRateLimitMiddleware(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Identify user (by IP in this example, you could use other identifiers)
		userIP := c.ClientIP()

		// Check last request time
		mu.Lock()
		lastRequestTime, exists := userRequestTimes[userIP]
		mu.Unlock()

		if exists && time.Since(lastRequestTime) < (duration*time.Second) {
			// If the user revisits within 60s, return 429 Too Many Requests
			c.JSON(http.StatusOK, gin.H{
				"code":  http.StatusTooManyRequests,
				"error": fmt.Sprintf("Too many requests. Please wait %d seconds before retrying.", duration),
			})
			c.Abort()
			return
		}

		// Update the last request time
		mu.Lock()
		userRequestTimes[userIP] = time.Now()
		mu.Unlock()

		// Continue with the request
		c.Next()
	}
}
