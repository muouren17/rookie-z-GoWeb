package mysql

import (
	"database/sql"
	"fmt"
	"strings"
	"webapp/models"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func GetCommunityList() (data []*models.Community, err error) {
	data = make([]*models.Community, 0)
	sqlstr := "select community_id,community_name from community"
	if err = db.Select(&data, sqlstr); err != nil {
		if err == sql.ErrNoRows {
			zap.L().Warn("no community in db")
			err = ErrorNoCorelateInfo
		}
	}

	return
}

func GetCommunityDetailByID(id int64) (data *models.CommunityDetail, err error) {
	data = new(models.CommunityDetail)
	sqlstr := "select community_id , community_name, introduction, create_time from community where community_id=?"
	if err = db.Get(data, sqlstr, id); err != nil {
		if err == sql.ErrNoRows {
			zap.L().Warn("no data corelated")
			err = ErrorInvalidID
		}
	}
	return
}

func CreatePost(p *models.Post) (err error) {
	sqlstr := "insert into post(post_id,author_id,community_id,status,title,content,create_time) values(:post_id,:author_id,:community_id,:status,:title,:content,:create_time)"
	_, err = db.NamedExec(sqlstr, p)
	return
}

func GetPostDetail(id int64) (data *models.PostDetail, err error) {
	data = new(models.PostDetail)
	sqlstr := "select post_id,author_id,community_id,title,content,create_time from post where post_id = ?"
	if err = db.Get(data, sqlstr, id); err != nil {
		if err == sql.ErrNoRows {
			warninfo := fmt.Sprintf("未能查到序号为%d的帖子", id)
			zap.L().Warn(warninfo, zap.Error(err))
			err = ErrorNoCorelateInfo
			return
		}
		err = ErrorInvalidID
	}
	return
}

func GetPostList(page, size int64) (posts []*models.PostDetail, err error) {
	posts = make([]*models.PostDetail, 0)
	sqlstr := "select post_id,author_id,community_id,title,content,create_time from post order by create_time desc limit ?,?"
	if err = db.Select(&posts, sqlstr, (page-1)*size, size); err != nil {
		err = ErrorNoCorelateInfo
		return
	}
	return
}

func GetPostListByIDs(ids []string) (post_list []*models.PostDetail, err error) {
	sqlstr := "select post_id,author_id,community_id,title,content,create_time from post where post_id in (?) order by FIND_IN_SET(post_id,?)"
	querys, args, err := sqlx.In(sqlstr, ids, strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}
	querys = db.Rebind(querys)
	err = db.Select(&post_list, querys, args...)
	return
}
