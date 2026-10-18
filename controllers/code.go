package controllers

type ResCode int64

const (
	CodeSuccess ResCode = 1000 + iota
	CodeInvalidParam
	CodeUserExist
	CodeUserNotExist
	CodeInvalidPassword
	CodeServerBusy

	CodeInvalidAuth
	CodeNeedLogin
	CodeStoreRedisFail
	CodeLoginFailed
	CodeFetchInfoFailed
	NotCode
)

var codeMsgMap = map[ResCode]string{
	CodeSuccess:         "success",
	CodeInvalidParam:    "请求参数错误",
	CodeUserExist:       "用户已存在",
	CodeUserNotExist:    "用户不存在",
	CodeInvalidPassword: "用户名或密码错误",
	CodeServerBusy:      "服务繁忙",
	NotCode:             "不存在这样的信息码",
	CodeInvalidAuth:     "无效的token",
	CodeNeedLogin:       "需要登录",
	CodeStoreRedisFail:  "存入Redis失败",
	CodeLoginFailed:     "该账户有用户登录，请重新登录",
	CodeFetchInfoFailed: "获取信息失败，请重试",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		return codeMsgMap[NotCode]
	}
	return msg
}
