package auth

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowCredentials = true
	// 当 AllowCredentials 为 true 时，不能使用 AllowAllOrigins = true（因为不能返回 *）
	// 使用 AllowOriginFunc 来动态允许所有源，但返回具体的 Origin 值
	config.AllowOriginFunc = func(origin string) bool {
		return true // 允许所有源
	}
	config.AddAllowHeaders("Authorization", "Accept", "Accept-Language")
	config.AddExposeHeaders("Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "Accept-Language")
	return cors.New(config)
}
