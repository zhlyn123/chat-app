package middleware

import (
	"chat-app/internal/application/service"
	"chat-app/internal/infrastructure/config"
	"chat-app/internal/infrastructure/logger"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Logger.Warn("jwt missing", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Logger.Warn("jwt malformed", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		if service.IsTokenBlacklisted(parts[1]) {
			logger.Logger.Warn("jwt blacklisted", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return config.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			logger.Logger.Warn("jwt auth failed", "path", c.Request.URL.Path, "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			logger.Logger.Warn("jwt claims invalid", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			logger.Logger.Warn("jwt user_id invalid", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			logger.Logger.Warn("jwt username invalid", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", uint(userIDFloat))
		c.Set("username", username)
		c.Set("token", parts[1])

		c.Next()
	}
}
