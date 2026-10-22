package logic

import (
	"context"
	"time"
	"webapp/dao/mysql"
	"webapp/dao/redis"
	"webapp/models"
	"webapp/pkg/snowflake"

	"go.uber.org/zap"
)

func GetCommunityList() (data []*models.Community, err error) {
	// 查找所有的community的id和信息并返回
	data, err = mysql.GetCommunityList()
	return
}

func GetCommunityDetail(id int64) (data *models.CommunityDetail, err error) {
	data, err = mysql.GetCommunityDetailByID(id)
	return
}

func CreatePost(p *models.ParamPost) (err error) {
	u := &models.Post{
		ID:          snowflake.GenID(),
		AuthorID:    p.AuthorID,
		CommunityID: p.CommunityID,
		Status:      1,
		Title:       p.Title,
		Content:     p.Content,
		CreateTime:  time.Now(),
	}
	err = mysql.CreatePost(u)
	if err != nil {
		return
	}
	err = redis.CreatePost(u.ID, u.CommunityID)
	return
}

func GetPostDetail(id int64) (data *models.ApiPostDetail, err error) {
	data = new(models.ApiPostDetail)
	dataPost, err := mysql.GetPostDetail(id)
	if err != nil {
		zap.L().Error("mysql.GetPostDetail failed ", zap.Error(err))
		return
	}
	datauser, err := mysql.GetUserDetail(dataPost.AuthorID)
	if err != nil {
		zap.L().Error("mysql.GetUserDetail failed ", zap.Error(err))
		return
	}
	datacommunity, err := GetCommunityDetail(dataPost.CommunityID)
	if err != nil {
		zap.L().Error("GetCommunityDetail failed ", zap.Error(err))
		return
	}
	data.AuthorName = datauser.Username
	data.PostDetail = dataPost
	data.CommunityDetail = datacommunity
	return
}

func GetPostList(page, size int64) (datas []*models.ApiPostDetail, err error) {
	posts, err := mysql.GetPostList(page, size)
	if err != nil {
		zap.L().Error("mysql.GetPostList failed ", zap.Error(err))
		return
	}

	datas = make([]*models.ApiPostDetail, 0, len(posts))
	for _, post := range posts {
		datauser, err := mysql.GetUserDetail(post.AuthorID)
		if err != nil {
			zap.L().Error("mysql.GetUserDetail failed ", zap.Error(err))
			continue
		}
		datacommunity, err := GetCommunityDetail(post.CommunityID)
		if err != nil {
			zap.L().Error("GetCommunityDetail failed ", zap.Error(err))
			continue
		}
		data := &models.ApiPostDetail{
			AuthorName:      datauser.Username,
			PostDetail:      post,
			CommunityDetail: datacommunity,
		}
		datas = append(datas, data)
	}
	return
}

func GetPostList2(ctx context.Context, p *models.ParamPostList) (datas []*models.ApiPostDetail, err error) {
	ids, err := redis.GetPostIDsInorder(ctx, p)
	if err != nil {
		return
	}
	if len(ids) == 0 {
		zap.L().Warn("redis.GetPostIDsInorder(p) return 0 data")
		return
	}

	scores, err := redis.GetPostVoteData(ctx, ids)
	if err != nil {
		return
	}

	posts := make([]*models.PostDetail, 0, len(ids))
	//GetPostListByIDs
	posts, err = mysql.GetPostListByIDs(ids)
	if err != nil {
		return nil, err
	}
	// for _, idstr := range ids {
	// 	id, err := strconv.ParseInt(idstr, 10, 64)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	temp, err := mysql.GetPostDetail(id)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	posts = append(posts, temp)
	// }

	datas = make([]*models.ApiPostDetail, 0, len(posts))
	for idx, post := range posts {
		datauser, err := mysql.GetUserDetail(post.AuthorID)
		if err != nil {
			zap.L().Error("mysql.GetUserDetail failed ", zap.Error(err))
			continue
		}
		datacommunity, err := GetCommunityDetail(post.CommunityID)
		if err != nil {
			zap.L().Error("GetCommunityDetail failed ", zap.Error(err))
			continue
		}
		data := &models.ApiPostDetail{
			AuthorName:      datauser.Username,
			Votenum:         scores[idx],
			PostDetail:      post,
			CommunityDetail: datacommunity,
		}
		datas = append(datas, data)
	}
	return

}

func GetCommunityPostList(ctx context.Context, p *models.ParamPostList) (datas []*models.ApiPostDetail, err error) {
	ids, err := redis.GetCommunityPostIDsInOrder(ctx, p)
	if err != nil {
		return
	}
	if len(ids) == 0 {
		zap.L().Warn("redis.GetCommunityPostIDsInOrder(p) return 0 data")
		return
	}

	scores, err := redis.GetPostVoteData(ctx, ids)
	if err != nil {
		return
	}

	posts := make([]*models.PostDetail, 0, len(ids))
	//GetPostListByIDs
	posts, err = mysql.GetPostListByIDs(ids)
	if err != nil {
		return nil, err
	}
	// for _, idstr := range ids {
	// 	id, err := strconv.ParseInt(idstr, 10, 64)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	temp, err := mysql.GetPostDetail(id)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	posts = append(posts, temp)
	// }

	datas = make([]*models.ApiPostDetail, 0, len(posts))
	for idx, post := range posts {
		datauser, err := mysql.GetUserDetail(post.AuthorID)
		if err != nil {
			zap.L().Error("mysql.GetUserDetail failed ", zap.Error(err))
			continue
		}
		datacommunity, err := GetCommunityDetail(post.CommunityID)
		if err != nil {
			zap.L().Error("GetCommunityDetail failed ", zap.Error(err))
			continue
		}
		data := &models.ApiPostDetail{
			AuthorName:      datauser.Username,
			Votenum:         scores[idx],
			PostDetail:      post,
			CommunityDetail: datacommunity,
		}
		datas = append(datas, data)
	}
	return
}

func GetPostListNew(ctx context.Context, p *models.ParamPostList) (datas []*models.ApiPostDetail, err error) {
	if p.CommunityID == 0 {
		datas, err = GetPostList2(ctx, p)
	} else {
		datas, err = GetCommunityPostList(ctx, p)
	}
	if err != nil {
		zap.L().Error("GetPostListNew failed ", zap.Error(err))
		return nil, err
	}

	return
}
