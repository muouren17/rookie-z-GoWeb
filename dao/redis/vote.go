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

var voteScript = redis.NewScript(`
-- KEYS:
-- 1 发布时间集合
-- 2 热度集合
-- 3 当前帖子的投票集合
-- 4 关闭帖子的集合

-- ARGV:
-- 1 帖子ID，2 用户ID，3 投票方向
-- 4 当前时间，5 投票有效期，6 每票分数

local postID = ARGV[1]
local userID = ARGV[2]
local direction = tonumber(ARGV[3])

-- 查询发布时间
local postTime = redis.call("ZSCORE", KEYS[1], postID)
if not postTime then
    return -2
end

-- 判断是否超过投票期限
if redis.call("SISMEMBER",KEYS[4],postID) ==1 then
	return -1
end

-- 写入前检查热度索引
if not redis.call("ZSCORE", KEYS[2], postID) then
    return -3
end

-- 查询用户旧的投票方向
local oldValue = redis.call("ZSCORE", KEYS[3], userID)
local oldDirection = 0
if oldValue then
    oldDirection = tonumber(oldValue)
end

-- 计算热度变化
local change = (direction - oldDirection) * tonumber(ARGV[6])
if change == 0 then
    return 0
end

-- 更新全局热度、用户投票方向
redis.call("ZINCRBY", KEYS[2], change, postID)
redis.call("ZADD", KEYS[3], direction, userID)

return 1
`)

var (
	ErrorVoteTimeExpire = errors.New("超出投票时间")
	ErrorRedisSetFailed = errors.New("分数变更失败")
	ErrorVoteRepeated   = errors.New("不允许重复投票")
	ErrorNoSuchPost     = errors.New("没有相关帖子")
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

	pipeline.ZAdd(ctx, KeyPostStorePendingZSet, redis.Z{
		Score:  float64(OneWeekInSeconds + time.Now().Unix()),
		Member: postID,
	})

	pipeline.SAdd(ctx, KeyCommunitySet+":"+strconv.Itoa(int(communityID)), postID)
	_, err = pipeline.Exec(ctx)
	return
}

func VoteForPost(id int64, p *models.ParamVoteData) (err error) {
	if p == nil || id <= 0 || p.PostID <= 0 ||
		p.Direction < -1 || p.Direction > 1 {
		return errors.New("无效的投票参数")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	postID := strconv.Itoa(int(p.PostID))
	userID := strconv.Itoa(int(id))
	voteKey := KeyPostVotedZSet + ":" + postID

	result, err := voteScript.Run(
		ctx,
		rdb,
		[]string{
			KeyPostTimeZSet,
			KeyPostScoreZSet,
			voteKey,
			KeyPostClosedSet,
		},
		postID,
		userID,
		p.Direction,
		time.Now().Unix(),
		OneWeekInSeconds,
		ScorePerVote,
	).Int64()
	if err != nil {
		return fmt.Errorf("%w:%v", ErrorRedisSetFailed, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return ErrorVoteRepeated
	case -1:
		return ErrorVoteTimeExpire
	case -2:
		return ErrorNoSuchPost
	case -3:
		return ErrorNoSuchPost
	default:
		return fmt.Errorf("未知的投票结果:%d", result)

	}

}

// func VoteForPost(id int64, p *models.ParamVoteData) (err error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
// 	defer cancel()
// 	postTime := rdb.ZScore(ctx, KeyPostTimeZSet, fmt.Sprintf("%d", p.PostID)).Val()
// 	if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
// 		err = ErrorVoteTimeExpire
// 		return
// 	}

// 	temp := rdb.ZScore(ctx, fmt.Sprintf("%s:%d", KeyPostVotedZSet, p.PostID), fmt.Sprintf("%d", id)).Val()
// 	scorechange := (float64(p.Direction) - temp) * ScorePerVote
// 	if scorechange == 0 {
// 		return ErrorVoteRepeated
// 	}

// 	pipeline := rdb.TxPipeline()

// 	pipeline.ZIncrBy(ctx, KeyPostScoreZSet, scorechange, fmt.Sprintf("%d", p.PostID))

// 	pipeline.ZAdd(ctx, fmt.Sprintf("%s:%d", KeyPostVotedZSet, p.PostID), redis.Z{
// 		Score:  float64(p.Direction),
// 		Member: fmt.Sprintf("%d", id),
// 	})

// 	_, err = pipeline.Exec(ctx)
// 	if err != nil {
// 		err = ErrorRedisSetFailed
// 	}
// 	return
// }
