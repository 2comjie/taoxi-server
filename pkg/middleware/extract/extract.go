package extract

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/gin-gonic/gin"
)

const maxClientRequestBodyBytes int64 = 1 << 20

func ClientExtract() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxClientRequestBodyBytes)
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			message := "请求解析失败"
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				message = "请求体过大"
			}
			xhttp.Fail(c, http.StatusBadRequest, message, nil)
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		format, err := midef.ParseRequestFormat(c.GetHeader("format"))
		if err != nil {
			xhttp.Fail(c, http.StatusBadRequest, "不支持的请求格式", nil)
			c.Abort()
			return
		}

		// 从HTTP header中提取参数
		request := midef.NewHeader(c.Request.Context())
		switch format {
		case midef.HeaderFromHeaderAndBodyFromBody:
			if err = c.ShouldBindHeader(request); err != nil {
				xhttp.Fail(c, http.StatusBadRequest, "解析请求头失败", err.Error())
				c.Abort()
				return
			}
			request.Body = string(bodyBytes)
		case midef.AllInForm:
			if err = c.ShouldBind(request); err != nil {
				xhttp.Fail(c, http.StatusBadRequest, "解析请求参数失败", err.Error())
				c.Abort()
				return
			}
		default:
			xhttp.Fail(c, http.StatusBadRequest, "不支持的请求格式", nil)
			c.Abort()
			return
		}

		// IP和请求格式以服务端解析结果为准，不能被客户端参数覆盖
		request.IP = c.ClientIP()
		request.RequestFormat = format
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		midef.SetClientRequestHeader(c, request)
		c.Next()
	}
}
