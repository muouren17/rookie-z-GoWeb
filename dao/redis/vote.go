package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"webapp/models"

	"github.com/redis/go-redis/v9"
)

const (
	OneWeekInSeconds = 7 * 24 * 3600
	ScorePerVote     = 432
)

var (
	ErrorVoteTimeExpire = errors.New("超出投票时间")
	ErrorRedisSetFailed = errors.New("分数变更失败")
	ErrorVoteRepeated   = errors.New("不允许重复投票")
)

func CreatePost(postID, communityID int64) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	pipeline := rdb.TxPipeline()
	pipeline.ZAdd(ctx, KeyPostTimeZSet, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	pipeline.ZAdd(ctx, KeyPostScoreZSet, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	pipeline.SAdd(ctx, KeyCommunitySet+":"+strconv.Itoa(int(communityID)), postID)
	_, err = pipeline.Exec(ctx)
	return
}

func VoteForPost(id int64, p *models.ParamVoteData) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	postTime := rdb.ZScore(ctx, KeyPostTimeZSet, fmt.Sprintf("%d", p.PostID)).Val()
	if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
		err = ErrorVoteTimeExpire
		return
	}

	temp := rdb.ZScore(ctx, fmt.Sprintf("%s:%d", KeyPostVotedZSet, p.PostID), fmt.Sprintf("%d", id)).Val()
	scorechange := (float64(p.Direction) - temp) * ScorePerVote
	if scorechange == 0 {
		return ErrorVoteRepeated
	}

	pipeline := rdb.TxPipeline()

	pipeline.ZIncrBy(ctx, fmt.Sprintf("%s:%d", KeyPostScoreZSet, p.PostID), scorechange, fmt.Sprintf("%d", p.PostID))

	pipeline.ZAdd(ctx, fmt.Sprintf("%s:%d", KeyPostVotedZSet, p.PostID), redis.Z{
		Score:  float64(p.Direction),
		Member: fmt.Sprintf("%d", id),
	})

	_, err = pipeline.Exec(ctx)
	if err != nil {
		err = ErrorRedisSetFailed
	}
	return
}
