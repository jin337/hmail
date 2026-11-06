package middleware

import (
	"errors"
	"strings"

	"email-server/config"
	"email-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth 校验 Authorization 头（也兼容 query 参数），并把邮箱/密码放入上下文。
// 可选 hooks 在鉴权成功后调用（例如按用户懒加载恢复定时任务），不得阻塞请求。
func JWTAuth(secret string, hooks ...func(email, pwd string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			tokenStr = c.Query("Authorization")
		}
		tokenStr = strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer "))

		if tokenStr == "" {
			abort(c, "未登录，请先登录获取Token")
			return
		}

		claims := &model.UserClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil {
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				abort(c, "Token已过期，请重新登录")
			case errors.Is(err, jwt.ErrTokenNotValidYet):
				abort(c, "Token尚未生效")
			default:
				abort(c, "Token无效: "+err.Error())
			}
			return
		}
		if !token.Valid {
			abort(c, "Token无效或已过期")
			return
		}

		c.Set("userEmail", claims.Email)
		c.Set("userPwd", claims.Password)
		for _, hook := range hooks {
			if hook != nil {
				hook(claims.Email, claims.Password)
			}
		}
		c.Next()
	}
}

// abort 返回 401 并终止后续处理。
func abort(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": msg})
}

// FromConfig 使用当前配置中的密钥构造中间件。
func FromConfig() gin.HandlerFunc {
	return JWTAuth(config.Current().JWT.Secret)
}
