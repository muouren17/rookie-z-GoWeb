package controllers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "user_id"

var ErrorUserNotLogin = errors.New("用户未登录")

func getCurrentUser(c *gin.Context) (int64, error) {
	uid, ok := c.Get(ContextUserID)
	if !ok {
		return 0, ErrorUserNotLogin
	}
	userID, ok := uid.(int64)
	if !ok {
		return 0, ErrorUserNotLogin
	}
	return userID, nil

}

func getPageInfo(c *gin.Context) (page, size int64) {
	pagestr := c.Query("page")
	sizestr := c.Query("size")
	page, err := strconv.ParseInt(pagestr, 10, 64)
	if err != nil {
		page = 1
	}
	size, err = strconv.ParseInt(sizestr, 10, 64)
	if err != nil {
		size = 10
	}
	return
}
