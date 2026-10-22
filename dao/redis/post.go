package redis

import (
	"context"
	"strconv"
	"time"
	"webapp/models"

	"github.com/redis/go-redis/v9"
)

func getIDsFormKey(ctx context.Context, key string, page, size int64) ([]string, error) {
	start := (page - 1) * size
	end := start + size - 1
	// ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	// defer cancel()
	return rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  end,
		Rev:   true,
	}).Result()
}

func GetPostIDsInorder(ctx context.Context, p *models.ParamPostList) ([]string, error) {
	key := KeyPostTimeZSet
	if p.Order == models.OrderScore {
		key = KeyPostScoreZSet
	}
	return getIDsFormKey(ctx, key, p.Page, p.Size)

}

func GetPostDownData(ctx context.Context, postids []string) (downcount []int64, err error) {

	pipeline := rdb.Pipeline()
	for _, id := range postids {
		key := KeyPostVotedZSet + ":" + id
		pipeline.ZCount(ctx, key, "-1", "-1")

	}
	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return
	}
	for _, cmder := range cmders {
		v := cmder.(*redis.IntCmd).Val()
		downcount = append(downcount, v)
	}

	return
}

// GetPostUpData
func GetPostVoteData(ctx context.Context, ids []string) (scores []int64, err error) {

	// for _, id := range ids {
	// 	key := fmt.Sprintf("%s:%s", KeyPostVotedZSet, id)
	// 	score, err := rdb.ZCount(ctx, key, "1", "1").Result()
	// 	if err != nil {
	// 		continue
	// 	}
	// 	scores = append(scores, score)
	// }
	pipeline := rdb.Pipeline()
	for _, id := range ids {
		key := KeyPostVotedZSet + ":" + id
		pipeline.ZCount(ctx, key, "1", "1")

	}
	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return
	}
	for _, cmder := range cmders {
		v := cmder.(*redis.IntCmd).Val()
		scores = append(scores, v)
	}

	return
}

func GetCommunityPostIDsInOrder(ctx context.Context, p *models.ParamPostList) ([]string, error) {
	orderkey := KeyPostTimeZSet
	if p.Order == models.OrderScore {
		orderkey = KeyPostScoreZSet
	}
	ckey := KeyCommunitySet + ":" + strconv.Itoa(int(p.CommunityID))
	key := orderkey + ":" + strconv.Itoa(int(p.CommunityID))
	// ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	// defer cancel()
	if rdb.Exists(ctx, key).Val() < 1 {
		// 缓存key不存在
		pipeline := rdb.Pipeline()
		keys := []string{
			ckey,
			orderkey,
		}
		pipeline.ZInterStore(ctx, key, &redis.ZStore{
			Keys:      keys,
			Aggregate: "MAX",
		})
		pipeline.Expire(ctx, key, time.Second*60)
		_, err := pipeline.Exec(ctx)
		if err != nil {
			return nil, err
		}
	}
	return getIDsFormKey(ctx, key, p.Page, p.Size)
}

func GetPostScore(ctx context.Context, postID string) (score float64, err error) {

	score, err = rdb.ZScore(ctx, KeyPostScoreZSet, postID).Result()
	return
}
