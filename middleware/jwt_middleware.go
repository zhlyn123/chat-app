package middleware

import (
	"chat-app/config"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth 中间件，保护路由
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 token，格式：Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		fmt.Println(len(parts),"parts0"+parts[0],"parts1"+parts[1])
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		// 解析 token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return config.JWTSecret, nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "token 无效"})
            c.Abort()
            return
        }

        // 将 user 信息存到 context 里，后续 handler 可以用
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "token claims 错误"})
            c.Abort()
            return
        }

        c.Set("user_id", uint(claims["user_id"].(float64))) // jwt 会解析成 float64
        c.Set("username", claims["username"].(string))

        c.Next()
	}
}
