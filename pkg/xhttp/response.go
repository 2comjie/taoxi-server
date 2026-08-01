package xhttp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Response(code int, message string, data any) map[string]any {
	result := make(map[string]interface{})
	result["code"] = code
	result["msg"] = message
	result["result"] = data
	return result
}

func Ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response(http.StatusOK, "ok", data))
}

func Fail(c *gin.Context, code int, message string, data any) {
	c.JSON(http.StatusOK, Response(code, message, data))
}
