package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Printf("Error processing request: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "internal_server_error",
				"message": "An internal error occurred",
			})
		}
	}
}