package logic

import (
	"webapp/dao/mysql"
	"webapp/dao/redis"
	"webapp/models"
)

func PostVote(id int64, p *models.ParamVoteData) (err error) {
	// 判断帖子是否存在
	if _, err = mysql.GetPostDetail(p.PostID); err != nil {
		return
	}

	// 更新帖子分数
	err = redis.VoteForPost(id, p)

	// 将用户进行记录

	return

}
