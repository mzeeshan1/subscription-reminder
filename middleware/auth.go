package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"subscription-manager/cache"
	"subscription-manager/config"
)

type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func Auth(cfg *config.Config, c *cache.Cache) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenStr, err := ctx.Cookie("token")
		if err != nil || tokenStr == "" {
			ctx.Redirect(http.StatusFound, "/login")
			ctx.Abort()
			return
		}

		claims := &Claims{}
		parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !parsed.Valid {
			ctx.Redirect(http.StatusFound, "/login")
			ctx.Abort()
			return
		}

		blacklisted, err := c.IsTokenBlacklisted(ctx.Request.Context(), tokenStr)
		if err != nil || blacklisted {
			ctx.Redirect(http.StatusFound, "/login")
			ctx.Abort()
			return
		}

		ctx.Set("userID", claims.UserID)
		ctx.Set("email", claims.Email)
		ctx.Set("token", tokenStr)
		ctx.Next()
	}
}
