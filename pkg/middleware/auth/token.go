package auth

import (
	"crypto/ed25519"
	"net/http"
	"strings"

	"github.com/2comjie/taoxi-server/pkg/jwt"
	midef "github.com/2comjie/taoxi-server/pkg/middleware/def"
	"github.com/2comjie/taoxi-server/pkg/xhttp"
	"github.com/gin-gonic/gin"
)

func AccessToken(publicKey ed25519.PublicKey) gin.HandlerFunc {
	key := append(ed25519.PublicKey(nil), publicKey...)
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.Next()
			return
		}
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			xhttp.Fail(c, http.StatusUnauthorized, "访问令牌格式无效", nil)
			c.Abort()
			return
		}
		uid, err := jwt.AuthAccessToken(key, []byte(parts[1]))
		if err != nil {
			xhttp.Fail(c, http.StatusUnauthorized, "访问令牌无效或已过期", nil)
			c.Abort()
			return
		}
		header, ok := midef.GetClientRequestHeader(c)
		if !ok {
			xhttp.Fail(c, http.StatusBadRequest, "获取http头失败", nil)
			c.Abort()
			return
		}
		header.UID = uid
		c.Next()
	}
}
