package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type apiData map[string]any

func respondOK(c *gin.Context, data apiData) {
	if data == nil {
		data = apiData{}
	}
	c.JSON(http.StatusOK, apiData{
		"success": true,
		"data":    data,
	})
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, apiData{
		"success": false,
		"error":   message,
	})
}
