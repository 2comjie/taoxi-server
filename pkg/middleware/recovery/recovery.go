package recovery

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/2comjie/nova/logx"
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
