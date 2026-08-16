package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/2comjie/nova/logx"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

const (
	SignKey = "Rjxj4rBKoXHeQZeeScHBT7fnYI6n4Ij8"
)

var (
	nonceCache = cache.New(5*time.Minute, 30*time.Second)
)

func CheckClientSign() gin.HandlerFunc {
	return func(c *gin.Context) {
		header, ok := midef.GetClientRequestHeader(c)
		if !ok {
			logx.WithField("url", c.Request.RequestURI).Error("获取http头失败")
			xhttp.Fail(c, http.StatusBadRequest, "获取http头失败", nil)
			c.Abort()
			return
		}
		if header.RequestFormat == midef.HeaderFromHeaderAndBodyFromBody {
			c.Next()
			return
		}

		accessToken := header.Sign
		if len(accessToken) == 0 {
			xhttp.Fail(c, http.StatusForbidden, "未提供access token", nil)
			c.Abort()
			return
		}

		uri := c.Request.URL.String()[1:]
		if c.Request.ContentLength > 10000 {
			logx.Infof("post data too large: url=%s, content-length=%v", uri, c.Request.Form)
		}

		if !checkNonceValid(header.Nonce) {
			xhttp.Fail(c, http.StatusBadRequest, "重复请求", nil)
			c.Abort()
			return
		}

		signParams := []string{"token", "lang", "nonce"}
		buf := bytes.NewBuffer(nil)
		for i, param := range signParams {
			if i > 0 {
				buf.WriteString("&")
			}
			var value string
			switch param {
			case "token":
				value = header.Token
			case "lang":
				value = header.Lang
			case "nonce":
				value = header.Nonce
			}
			buf.WriteString(fmt.Sprintf("%s=%s", param, value))
		}
		buf.WriteString(fmt.Sprintf("&body=%s", base64.StdEncoding.EncodeToString([]byte(header.Body))))
		hexString, _ := hex.DecodeString(Hmac(SignKey, buf.String()))
		base64String := base64.StdEncoding.EncodeToString(hexString)

		if base64String != accessToken {
			logx.WithField("url", c.Request.RequestURI).Warn("签名不正确")
			xhttp.Fail(c, http.StatusBadRequest, "签名校验失败", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func checkNonceValid(nonce string) bool {
	if nonce == "" {
		return true
	}
	err := nonceCache.Add(nonce, struct{}{}, time.Minute*5)
	if err != nil {
		return false
	}
	return true
}

func Hmac(key, data string) string {
	hmac := hmac.New(sha1.New, []byte(key))
	hmac.Write([]byte(data))
	return hex.EncodeToString(hmac.Sum(nil))
}
