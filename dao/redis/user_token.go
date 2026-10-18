package redis

import (
	"context"
	"strconv"
	"time"
	"webapp/models"
)

func StoreToken(p *models.Pair, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := rdb.Set(ctx, strconv.FormatInt(p.UserID, 10), p.Token, ttl).Err()
	return err
}

func GetToken(p int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	value, err := rdb.Get(ctx, strconv.FormatInt(p, 10)).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}
