package inout

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/stderr"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/2comjie/wali/logx"
	"github.com/gin-gonic/gin"
)

type StdFunc[Req any, Rsp any] func(ctx *midef.Header, req *Req) (*Rsp, *stderr.Error)

func UidHandler[Req any, Rsp any](handler StdFunc[Req, Rsp]) gin.HandlerFunc {
	return stdHandler(handler, true)
}

func NoUidHandler[Req any, Rsp any](handler StdFunc[Req, Rsp]) gin.HandlerFunc {
	return stdHandler(handler, false)
}

func stdHandler[Req any, Rsp any](handler StdFunc[Req, Rsp], checkUID bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		header, ok := midef.GetClientRequestHeader(c)
		if !ok {
			logx.WithField("url", c.Request.RequestURI).Error("获取http头失败")
			xhttp.Fail(c, http.StatusBadRequest, "获取http头失败", nil)
			c.Abort()
			return
		}
		if checkUID && header.UID == "" {
			xhttp.Fail(c, http.StatusUnauthorized, "未提供有效的访问令牌", nil)
			c.Abort()
			return
		}
		var req Req
		err := c.ShouldBind(&req)
		if err != nil {
			logx.WithField("url", c.Request.RequestURI).Error("解析请求参数失败")
			xhttp.Fail(c, http.StatusBadRequest, "请求解析失败", nil)
			c.Abort()
			return
		}
		rsp, stdErr := handler(header, &req)
		if stdErr != nil {
			xhttp.Fail(c, stdErr.Code, stdErr.Msg, rsp)
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
			logx.WithField("url", c.Request.RequestURI).Error("解析请求参数失败")
			xhttp.Fail(c, http.StatusBadRequest, "请求解析失败", nil)
			c.Abort()
			return
		}
		rsp, stdErr := handler(&req)
		if stdErr != nil {
			xhttp.Fail(c, stdErr.Code, stdErr.Msg, rsp)
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
			accessLogMap["post_data"] = redactForm(c.Request.PostForm).Encode()
			accessLogMap["bytes_send"] = c.Writer.Size()
			logx.Infof("access_log: %v", accessLogMap)
		}()
		c.Next()
	}
}

func redactForm(form url.Values) url.Values {
	redacted := make(url.Values, len(form))
	for key, values := range form {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") || lowerKey == "code" || lowerKey == "guest_id" {
			redacted[key] = []string{"[REDACTED]"}
			continue
		}
		redacted[key] = append([]string(nil), values...)
	}
	return redacted
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
