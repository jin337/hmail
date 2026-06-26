package middleware

import (
	"email-server/constant"
	"errors"

	"email-server/config"
	"email-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			tokenStr = c.Query("Authorization")
		}

		if tokenStr == "" {
			c.JSON(401, gin.H{"code": 401, "msg": "未登录，请先登录获取Token"})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &model.UserClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(config.GetConfig(constant.JwtSecret)), nil
		})
		if err != nil {
			// 检查是否是过期错误
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(401, gin.H{"code": 401, "msg": "Token已过期，请重新登录"})
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				c.JSON(401, gin.H{"code": 401, "msg": "Token尚未生效"})
			} else {
				c.JSON(401, gin.H{"code": 401, "msg": "Token无效: " + err.Error()})
			}
			c.Abort()
			return
		}

		if !token.Valid {
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
