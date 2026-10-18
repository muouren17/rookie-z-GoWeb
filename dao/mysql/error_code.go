package mysql

import "errors"

var (
	ErrorUserExist       = errors.New("用户已存在...")
	ErrorUserNotExits    = errors.New("用户不存在")
	ErrorInvalidPassword = errors.New("密码不正确，请重新输入密码")
	ErrorInvalidID       = errors.New("无效的ID")
	ErrorNoCorelateInfo  = errors.New("未查询到相关信息")
)
