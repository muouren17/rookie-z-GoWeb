package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func GettimeoutPostID(ctx context.Context) ([]string, error) {
	timenow := strconv.Itoa(int((time.Now().Unix())))
	results, err := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     KeyPostStorePendingZSet,
		Start:   "-inf",
		Stop:    "(" + timenow,
		ByScore: true,
		Offset:  0,
		Count:   100,
	}).Result()
	if err != nil {
		return nil, err
	}
	return results, nil
}

func SavetimeoutPostID(ctx context.Context, postIDs []string) (err error) {

	for _, postid := range postIDs {
		_, err = rdb.SAdd(ctx, KeyPostClosedSet, postid).Result()
		if err != nil {
			return
		}
	}
	return

}

func RemtimeoutPost(ctx context.Context, postIDs []string) (err error) {

	for _, postID := range postIDs {
		err = rdb.ZRem(ctx, KeyPostStorePendingZSet, postID).Err()
		if err != nil {
			return
		}
	}
	return
}
