package middleware

import (
	"avito-easy-report/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired проверяет JWT-токен и добавляет claims в контекст.
// Токен может быть в заголовке Authorization: Bearer <token> или в query-параметре ?token=<token>
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется авторизация"})
			return
		}

		claims, err := service.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

// AdminRequired требует роль admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "доступ запрещён"})
			return
		}

		cClaims := claims.(*service.Claims)
		if cClaims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "требуется роль администратора"})
			return
		}
		c.Next()
	}
}

// GetClaims извлекает claims из контекста
func GetClaims(c *gin.Context) *service.Claims {
	claims, exists := c.Get("claims")
	if !exists {
		return nil
	}
	return claims.(*service.Claims)
}
