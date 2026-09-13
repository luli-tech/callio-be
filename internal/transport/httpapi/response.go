package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Code       string
	Message    string
	StatusCode int
}

func WriteError(c *gin.Context, err Error) {
	c.JSON(err.StatusCode, gin.H{
		"code":    err.Code,
		"message": err.Message,
	})
}

func InvalidRequest(c *gin.Context, message string) {
	WriteError(c, Error{Code: "INVALID_REQUEST", Message: message, StatusCode: http.StatusBadRequest})
}

func Unauthorized(c *gin.Context, code, message string) {
	WriteError(c, Error{Code: code, Message: message, StatusCode: http.StatusUnauthorized})
}

func InternalError(c *gin.Context, message string) {
	WriteError(c, Error{Code: "INTERNAL_ERROR", Message: message, StatusCode: http.StatusInternalServerError})
}

func NotFound(c *gin.Context, code, message string) {
	WriteError(c, Error{Code: code, Message: message, StatusCode: http.StatusNotFound})
}
