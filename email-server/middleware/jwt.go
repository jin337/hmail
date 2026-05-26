package middleware

import (
	"strings"

	"email-server/config"
	"email-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, config.TokenPrefix) {
			c.JSON(401, gin.H{"code": 401, "msg": "未登录，请先登录获取Token"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, config.TokenPrefix)

		token, err := jwt.ParseWithClaims(tokenStr, &model.UserClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(config.JwtSecretKey), nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"code": 401, "msg": "Token无效或已过期"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*model.UserClaims)
		if !ok {
			c.JSON(401, gin.H{"code": 401, "msg": "Token信息解析失败"})
			c.Abort()
			return
		}

		c.Set("userEmail", claims.Email)
		c.Set("userPwd", claims.Password)
		c.Next()
	}
}
