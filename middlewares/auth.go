package middlewares

import (
	"strings"
	"webapp/controllers"
	"webapp/logic"
	"webapp/models"
	"webapp/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			controllers.ResponseError(c, controllers.CodeNeedLogin)
			c.Abort()
			return
		}
		parts := strings.SplitN(token, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			controllers.ResponseError(c, controllers.CodeNeedLogin)
			c.Abort()
			return
		}
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			controllers.ResponseError(c, controllers.CodeInvalidAuth)
			c.Abort()
			return
		}
		p := &models.ParamPair{
			UserID: mc.UserID,
			Token:  parts[1],
		}
		ifsame, err := logic.GetUToken(p)
		if err != nil {
			controllers.ResponseError(c, controllers.CodeServerBusy)
			c.Abort()
			return
		}
		if ifsame != true {
			controllers.ResponseError(c, controllers.CodeLoginFailed)
			c.Abort()
			return
		}
		c.Set(controllers.ContextUserID, mc.UserID)
		c.Next()
	}
}
