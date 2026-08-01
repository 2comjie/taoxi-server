package inout

import (
	"net/http"

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
