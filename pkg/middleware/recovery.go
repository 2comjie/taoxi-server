package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/2comjie/wali/logx"
	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				msg := fmt.Sprintf("请求panic path: %s, stack: %v, err: %v", c.Request.URL.Path, string(debug.Stack()), err)
				logx.Errorf(msg)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessLogMap := make(map[string]any)
		accessLogMap["method"] = c.Request.Method
		accessLogMap["protocol"] = c.Request.Proto
		accessLogMap["referer"] = c.Request.Referer()
		accessLogMap["client_ip"] = c.ClientIP()
		accessLogMap["uri"] = c.Request.URL.String()
		accessLogMap["host"] = c.Request.Host

		defer func() {
			_ = c.Request.ParseForm()
			accessLogMap["post_data"] = c.Request.PostForm.Encode()
			accessLogMap["bytes_send"] = c.Writer.Size()
			logx.Infof("access_log: %v", accessLogMap)
		}()
		c.Next()
	}
}
