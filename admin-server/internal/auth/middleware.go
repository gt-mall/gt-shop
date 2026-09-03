package auth

import "github.com/gin-gonic/gin"

// Middleware is the future authentication boundary. The demo keeps routes
// public until login and token storage are implemented.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
