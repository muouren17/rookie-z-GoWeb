package controllers

import (
	"strconv"
	"webapp/logic"
	"webapp/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CommunityHandler 获取社区列表。
// @Summary 获取社区列表
// @Description 返回所有社区的ID和名称，社区ID为数字。成功1000；获取失败1010。鉴权失败可能返回1005、1006、1007、1009。
// @Tags 社区相关接口
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData{data=[]models.Community} "社区列表；无数据时 data 为[]"
// @Router /api/v1/community [get]
func CommunityHandler(c *gin.Context) {
	data, err := logic.GetCommunityList()
	if err != nil {
		zap.L().Error("logic.GetCommunityList failed", zap.Error(err))
		ResponseError(c, CodeFetchInfoFailed)
		return
	}
	ResponseSuccess(c, data)
}

// CommunityDetailHandler 获取社区详情。
// @Summary 获取社区详情
// @Description 根据社区ID查询名称、介绍和创建时间；intro 为空时省略。成功1000；参数错误1001、获取失败1010。业务错误也返回 HTTP 200。
// @Tags 社区相关接口
// @Produce json
// @Param id path string true "社区ID，可从社区列表取得，例如1"
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData{data=models.CommunityDetail} "社区详情"
// @Router /api/v1/community/{id} [get]
func CommunityDetailHandler(c *gin.Context) {
	temp := c.Param("id")
	id, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		ResponseError(c, CodeInvalidParam)
		c.Abort()
		return
	}
	data, err := logic.GetCommunityDetail(id)
	if err != nil {
		zap.L().Error("logic.GetCommunityDetail failed", zap.Error(err))
		ResponseError(c, CodeFetchInfoFailed)
		c.Abort()
		return
	}
	ResponseSuccess(c, data)
}

// CommunityCreatePostHandler 创建帖子。
// @Summary 创建帖子
// @Description 作者ID从当前 access token 获取，客户端无需提交 author_id。community_id 为JSON数字且当前限制1到4。成功1000，省略data且不返回新帖子ID；参数错误1001、服务繁忙1005。执行会真实写入 MySQL 和 Redis。
// @Tags 帖子相关接口
// @Accept json
// @Produce json
// @Param object body models.ParamPost true "帖子信息，community_id、title、content 必填"
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData "成功时 code=1000，data 字段省略"
// @Router /api/v1/post [post]
func CommunityCreatePostHandler(c *gin.Context) {
	// 获取参数以及校验
	p := new(models.ParamPost)
	if err := c.ShouldBindJSON(&p); err != nil {
		ResponseError(c, CodeInvalidParam)
		c.Abort()
		return
	}
	userID, err := getCurrentUser(c)
	if err != nil {
		ResponseError(c, CodeNeedLogin)
		c.Abort()
		return
	}
	p.AuthorID = userID
	// 创建帖子
	if err := logic.CreatePost(p); err != nil {
		zap.L().Error("logic.CreatePost failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		c.Abort()
		return
	}
	// 返回响应
	ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情。
// @Summary 获取帖子详情
// @Description 使用帖子的业务ID，而不是数据库自增行ID。响应 id 和 author_id 为JSON字符串，community_id 为数字，社区信息位于 community 对象。此接口不查询票数，vote_num 省略。参数错误1001、获取失败1010。
// @Tags 帖子相关接口
// @Produce json
// @Param id path string true "帖子业务ID，使用列表返回的 id，例如10000000000000001"
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData{data=models.ApiPostDetail} "帖子详情"
// @Router /api/v1/post/{id} [get]
func GetPostDetailHandler(c *gin.Context) {
	// 获取id
	temp := c.Param("id")
	postID, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		zap.L().Error("get post detail with invalid param", zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		c.Abort()
		return
	}
	// 根据id从数据库中获取post详情
	data, err := logic.GetPostDetail(postID)
	if err != nil {
		zap.L().Error("logic.GetPostDetail failed", zap.Error(err))
		ResponseError(c, CodeFetchInfoFailed)
		c.Abort()
		return
	}
	// 返回结果
	ResponseSuccess(c, data)
}

// GetPostListHandler 获取 MySQL 分页帖子列表。
// @Summary 获取帖子列表（MySQL分页）
// @Description 按创建时间降序排列，仅处理 page、size，不处理 order 或 community_id，也不查询票数。缺省或无法解析的 page/size 分别回退为1/10；当前没有正数或上限校验。获取失败返回1010。
// @Tags 帖子相关接口
// @Produce json
// @Param page query integer false "页码，建议大于等于1" default(1)
// @Param size query integer false "每页数量，建议1到100" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData{data=[]models.ApiPostDetail} "帖子列表；无数据时 data 为[]"
// @Router /api/v1/posts [get]
func GetPostListHandler(c *gin.Context) {
	page, size := getPageInfo(c)

	// 获取communityid
	data, err := logic.GetPostList(page, size)
	if err != nil {
		zap.L().Error("logic.GetPostList failed", zap.Error(err))
		ResponseError(c, CodeFetchInfoFailed)
		c.Abort()
		return
	}
	// 根据communityid获取所有的post
	// 返回结果
	ResponseSuccess(c, data)
}

// GetPostListHandler2 获取可排序、可按社区筛选的帖子列表。
// @Summary 获取帖子列表（排序和社区筛选）
// @Description 根据 Redis 索引排序并查询 MySQL。只有 order=score 时按热度降序，其余值按发布时间降序。community_id 省略或0查询全部，非0按社区筛选。社区排序结果缓存60秒。vote_num 为赞同票数，值为0时省略。分页数值解析失败返回1001；当前没有正数或上限校验；获取失败1010。
// @Tags 帖子相关接口
// @Produce json
// @Param page query integer false "页码，建议大于等于1" default(1)
// @Param size query integer false "每页数量，建议1到100；不要传0" default(10)
// @Param order query string false "time 按发布时间，score 按热度分数；其他值按时间处理" default(time)
// @Param community_id query integer false "社区ID，0查询全部，非0按社区筛选" default(0)
// @Security ApiKeyAuth
// @Success 200 {object} ResponseData{data=[]models.ApiPostDetail} "帖子列表；Redis无匹配ID时 data 为null"
// @Router /api/v1/posts2 [get]
func GetPostListHandler2(c *gin.Context) {
	p := &models.ParamPostList{
		Page:  1,
		Size:  10,
		Order: models.OrderTime,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler2 with invalid param", zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		c.Abort()
		return
	}

	data, err := logic.GetPostListNew(p)

	if err != nil {
		zap.L().Error("logic.GetPostList failed", zap.Error(err))
		ResponseError(c, CodeFetchInfoFailed)
		c.Abort()
		return
	}
	// 根据communityid获取所有的post
	// 返回结果
	ResponseSuccess(c, data)
}

// func GetCommunityPostHandler(c *gin.Context) {
// 	p := &models.ParamCommunityPostList{
// 		ParamPostList: models.ParamPostList{
// 			Page:  1,
// 			Size:  10,
// 			Order: models.OrderTime,
// 		},
// 	}

// 	if err := c.ShouldBindQuery(p); err != nil {
// 		zap.L().Error("GetPostListHandler2 with invalid param", zap.Error(err))
// 		ResponseError(c, CodeInvalidParam)
// 		c.Abort()
// 		return
// 	}

// 	data, err := logic.GetCommunityPostList(p)
// 	if err != nil {
// 		zap.L().Error("logic.GetPostList failed", zap.Error(err))
// 		ResponseError(c, CodeFetchInfoFailed)
// 		c.Abort()
// 		return
// 	}
// 	// 根据communityid获取所有的post
// 	// 返回结果
// 	ResponseSuccess(c, data)
// }
