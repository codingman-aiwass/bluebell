package middleware

import (
	"bluebell/controllers"
	"bluebell/logic"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strings"
)

func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			zap.L().Error("no auth header")
			controllers.ResponseError(c, controllers.CODE_NOT_LOGIN)
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			zap.L().Error("parse token error")
			controllers.ResponseError(c, controllers.CODE_INVALID_TOKEN)
			c.Abort()
			return
		}
		mc, err := logic.ParseToken(parts[1])
		if err != nil {
			zap.L().Error("parse token error", zap.Error(err))
			controllers.ResponseError(c, controllers.CODE_INVALID_TOKEN)
			c.Abort()
			return
		}
		// 判断目前token是否有多个用户登录
		if b, _ := logic.CheckMoreThanOneUser(mc.UserId, parts[1]); b {
			zap.L().Info("More than one user")
			controllers.ResponseError(c, controllers.CODE_MORE_THAN_ONE_USER)
			c.Abort()
			return
		}
		c.Set(controllers.ContextUserIdKey, mc.UserId)
		c.Set(controllers.ContextUserNameKey, mc.Username)
		c.Next()
	}
}
