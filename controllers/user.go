package controllers

import (
	"errors"
	"fmt"
	"webapp/dao/mysql"
	"webapp/logic"
	"webapp/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// SignUpHandler 用户注册。
// @Summary 用户注册
// @Description 注册新账号，password 与 repassword 必须相同。业务成功 code=1000；错误也返回 HTTP 200，参数错误1001、用户名已存在1002、服务繁忙1005。成功时省略 data。
// @Tags 账号与令牌
// @Accept json
// @Produce json
// @Param object body models.ParamSigUp true "注册信息，三个字段均必填"
// @Success 200 {object} ResponseData "通过 code 判断业务结果；msg 可能是错误字符串或字段校验错误对象"
// @Router /api/v1/signup [post]
func SignUpHandler(c *gin.Context) {
	p := new(models.ParamSigUp)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("SignUp with invalid param", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(c, CodeInvalidParam)
			return
		}
		ResponseErrorWithMsg(c, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	// 按字节检查；中文等字符在 UTF-8 中可能占用多个字节。
	if len(p.Password) > 72 {
		ResponseError(c, CodeInvalidParam)
		return
	}

	// if len(p.Username) == 0 || len(p.Password) == 0 || len(p.RePassword) == 0 || p.Password != p.RePassword {
	// 	zap.L().Error("SignUp with invalid param")
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"msg": "请求参数有误",
	// 	})
	// 	return
	// }

	if err := logic.SignUp(p); err != nil {
		zap.L().Error("logic.SignUp failed , error ", zap.Error(err))
		if errors.Is(err, mysql.ErrorUserExist) {
			ResponseError(c, CodeUserExist)
			return
		}
		ResponseError(c, CodeServerBusy)
		return
	}
	ResponseSuccess(c, nil)
}

// LoginHandler 用户登录。
// @Summary 用户登录
// @Description 登录后覆盖 Redis 中该用户的当前会话，旧设备后续访问会失效。返回字段实际拼写为 accessTokne 和 refreshToken，userID 是字符串。成功1000；参数错误1001、用户不存在1003、密码错误1004、服务繁忙1005。
// @Tags 账号与令牌
// @Accept json
// @Produce json
// @Param object body models.ParamLogin true "用户名和密码"
// @Success 200 {object} ResponseData{data=object{userID=string,username=string,accessTokne=string,refreshToken=string}} "登录结果；Authorize 中填写 Bearer 加一个空格再加 accessTokne"
// @Router /api/v1/login [post]
func LoginHandler(c *gin.Context) {
	p := new(models.ParamLogin)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("Login with invalid param", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(c, CodeInvalidParam)
			return
		}
		ResponseErrorWithMsg(c, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	if len(p.Password) > 72 {
		ResponseError(c, CodeInvalidParam)
		return
	}

	// 进行登录
	user, token, rtoken, err := logic.Login(p)
	if err != nil {
		zap.L().Error("logic.Login failed ", zap.Error(err))
		switch {
		case errors.Is(err, mysql.ErrorUserNotExits):
			ResponseError(c, CodeUserNotExist)
			return
		case errors.Is(err, mysql.ErrorInvalidPassword):
			ResponseError(c, CodeInvalidPassword)
			return
		default:
			ResponseError(c, CodeServerBusy)
		}
		return
	}

	// 返回登录结果
	ResponseSuccess(c, gin.H{
		"userID":       fmt.Sprintf("%d", user.UserID),
		"username":     user.Username,
		"accessTokne":  token,
		"refreshToken": rtoken,
	})
}

// RefreshHandler 刷新令牌。
// @Summary 刷新令牌
// @Description JSON示例：{"access_token":"替换为当前会话的access_token","refresh_token":"替换为配对的refresh_token"}
// @Description 根路径为 /refresh，不带 /api/v1。access_token 和 refresh_token 均必填且必须配对，使用当前会话的两个令牌。访问令牌未过期时返回原令牌对；只有访问令牌已过期且其他校验有效时才生成新令牌并更新 Redis。成功1000；参数错误1001、令牌校验失败1006。此接口不需要 Authorization。
// @Tags 账号与令牌
// @Accept json
// @Produce json
// @Param object body object{access_token=string,refresh_token=string} true "两个字段均必填，值为配对令牌；access_token 允许已过期"
// @Success 200 {object} ResponseData{data=object{access_token=string,refresh_token=string}} "刷新结果；下一次请求使用返回的 access_token"
// @Router /refresh [post]
func RefreshHandler(c *gin.Context) {
	var p struct {
		AccessToken  string `json:"access_token" binding:"required"`
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&p); err != nil {
		ResponseError(c, CodeInvalidParam)
		return
	}

	aToken, rToken, err := logic.RefreshToken(p.AccessToken, p.RefreshToken)
	if err != nil {
		zap.L().Error("RefreshToken failed", zap.Error(err))
		ResponseError(c, CodeInvalidAuth)
		return
	}

	ResponseSuccess(c, map[string]interface{}{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}
