package inout

import (
	"fmt"
	"net/http"
	"time"

	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/2comjie/wali/logx"
	"github.com/gin-gonic/gin"
)

func StdHandler[Req any, Rsp any](handler func(ctx *midef.Header, req *Req) (*Rsp, *stderr.Error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		header, ok := midef.GetClientRequestHeader(c)
		if !ok {
			logx.WithField("url", c.Request.RequestURI).WithField("header", c.Request.Header).Error("获取http头失败")
			xhttp.Fail(c, http.StatusBadRequest, "获取http头失败", nil)
			c.Abort()
			return
		}
		var req Req
		err := c.ShouldBind(&req)
		if err != nil {
			logx.WithField("url", c.Request.RequestURI).WithField("header", c.Request.Header).WithField("body", c.Request.Body).Error("解析请求参数失败")
			xhttp.Fail(c, http.StatusBadRequest, "请求解析失败", nil)
			c.Abort()
		}
		rsp, stdErr := handler(header, &req)
		if stdErr != nil {
			xhttp.Response(stdErr.Code, stdErr.Msg, rsp)
			return
		}
		xhttp.Ok(c, rsp)
	}
}

func NoHeadHandler[Req any, Rsp any](handler func(req *Req) (rsp *Rsp, stdErr *stderr.Error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Req
		err := c.ShouldBind(&req)
		if err != nil {
			logx.WithField("url", c.Request.RequestURI).WithField("header", c.Request.Header).WithField("body", c.Request.Body).Error("解析请求参数失败")
			xhttp.Fail(c, http.StatusBadRequest, "请求解析失败", nil)
			c.Abort()
		}
		rsp, stdErr := handler(&req)
		if stdErr != nil {
			xhttp.Response(stdErr.Code, stdErr.Msg, rsp)
			return
		}
		xhttp.Ok(c, rsp)
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

func LogxLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
		if params.StatusCode > 500 {
			logInfo := buildFullLogInfo(params)
			logx.Errorf("[HTTP] %s", logInfo)
		} else if params.StatusCode >= 400 {
			logInfo := buildClientErrorLogInfo(params)
			logx.Warnf("[HTTP] %s", logInfo)
		} else {
			logInfo := buildBasicLogInfo(params)
			logx.Infof("[HTTP] %s", logInfo)
		}
		return ""
	})
}

func buildFullLogInfo(param gin.LogFormatterParams) string {
	return fmt.Sprintf("timestamp=%s client_ip=%s status=%d latency=%s method=%s path=%s proto=%s body_size=%d user_agent=%s error=%s",
		param.TimeStamp.Format(time.RFC3339),
		param.ClientIP,
		param.StatusCode,
		param.Latency,
		param.Method,
		param.Path,
		param.Request.Proto,
		param.BodySize,
		param.Request.UserAgent(),
		param.ErrorMessage,
	)
}

func buildClientErrorLogInfo(param gin.LogFormatterParams) string {
	return fmt.Sprintf("status=%d method=%s path=%s latency=%s client_ip=%s error=%s",
		param.StatusCode,
		param.Method,
		param.Path,
		param.Latency,
		param.ClientIP,
		param.ErrorMessage,
	)
}

func buildBasicLogInfo(param gin.LogFormatterParams) string {
	return fmt.Sprintf("status=%d method=%s path=%s latency=%s",
		param.StatusCode,
		param.Method,
		param.Path,
		param.Latency,
	)
}
