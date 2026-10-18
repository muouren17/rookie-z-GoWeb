package controllers

import (
	"webapp/logic"
	"webapp/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// 用户id,postid,投票类型
// PostVoteHandler 赞同、反对或取消投票。
// @Summary 帖子投票
// @Description JSON示例：{"post_id":"10000000000000001","direction":"1"}
// @Description post_id 必填且不能为0；post_id、direction 都必须是JSON字符串。direction 为"1"赞同、"-1"反对、"0"取消，省略时当前按0处理。用户ID从access token获取。帖子超过7天不能投票。参数错误1001；帖子不存在、重复投票、超过投票期限或存储失败等当前统一返回1005。成功1000且省略data。执行会真实修改投票数据。
// @Tags 投票相关接口
// @Accept json
// @Produce json
// @Param object body object{post_id=string,direction=string} true "post_id 必填；direction 可省略，取字符串1、0或-1"
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData "通过 code 判断业务结果"
// @Router /api/v1/vote [post]
func PostVoteHandler(c *gin.Context) {
	v := new(models.ParamVoteData)
	if err := c.ShouldBindJSON(v); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			ResponseError(c, CodeInvalidParam)
			c.Abort()
			return
		}
		errData := removeTopStruct(errs.Translate(trans))
		ResponseErrorWithMsg(c, CodeInvalidParam, errData)
		c.Abort()
		return
	}

	userID, err := getCurrentUser(c)
	if err != nil {
		ResponseError(c, CodeNeedLogin)
		c.Abort()
		return
	}

	err = logic.PostVote(userID, v)
	if err != nil {
		zap.L().Error("logic.PostVote failed ", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		c.Abort()
		return
	}

	ResponseSuccess(c, nil)
}
