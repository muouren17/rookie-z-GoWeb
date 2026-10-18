package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseData 统一接口响应。业务成功code为1000；业务错误通常也返回HTTP 200。
type ResponseData struct {
	// Code 业务状态码：1000成功，1001参数错误，1002用户已存在，1003用户不存在，1004密码错误，1005服务繁忙，1006无效令牌，1007需要登录，1009会话被替换，1010获取失败。
	Code ResCode     `json:"code"`
	// Msg 成功时为success；失败时为错误字符串，字段校验失败时也可能是字段到错误信息的对象。
	Msg  interface{} `json:"msg"`
	// Data 响应数据，nil接口值时省略；包含类型的nil切片可能序列化为null。
	Data interface{} `json:"data,omitempty"`
}

func ResponseError(c *gin.Context, code ResCode) {
	rd := &ResponseData{
		Code: code,
		Msg:  code.Msg(),
		Data: nil,
	}
	c.JSON(http.StatusOK, rd)
}

func ResponseErrorWithMsg(c *gin.Context, code ResCode, msg interface{}) {
	rd := &ResponseData{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
	c.JSON(http.StatusOK, rd)
}

func ResponseSuccess(c *gin.Context, data interface{}) {
	rd := &ResponseData{
		Code: CodeSuccess,
		Msg:  CodeSuccess.Msg(),
		Data: data,
	}
	c.JSON(http.StatusOK, rd)
}
