package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the Gin router for the walking skeleton server.
func NewRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}

// WriteError writes a consistent error payload for the API.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
