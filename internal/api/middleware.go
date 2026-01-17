package api

import (
        "fmt"
        "time"

        "github.com/gin-gonic/gin"
)

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
        return gin.HandlerFunc(func(c *gin.Context) {
                c.Header("Access-Control-Allow-Origin", "*")
                c.Header("Access-Control-Allow-Credentials", "true")
                c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
                c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

                if c.Request.Method == "OPTIONS" {
                        c.AbortWithStatus(204)
                        return
                }

                c.Next()
        })
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
        return gin.HandlerFunc(func(c *gin.Context) {
                // Simple rate limiting logic can be implemented here
                // For now, just pass through
                c.Next()
        })
}

// LoggingMiddleware provides request logging
func LoggingMiddleware() gin.HandlerFunc {
        return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
                return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
                        param.ClientIP,
                        param.TimeStamp.Format(time.RFC1123),
                        param.Method,
                        param.Path,
                        param.Request.Proto,
                        param.StatusCode,
                        param.Latency,
                        param.Request.UserAgent(),
                        param.ErrorMessage,
                )
        })
}