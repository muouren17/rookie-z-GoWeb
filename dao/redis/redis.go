package redis

import (
	"context"
	"fmt"
	"time"
	"webapp/settings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var rdb *redis.Client

func Init(cfg *settings.RedisConfig) (cancel context.CancelFunc, err error) {
	rdb = redis.NewClient(&redis.Options{
		// Addr:     fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port")),
		// Password: viper.GetString("redis.password"),
		// DB:       viper.GetInt("redis.db"),
		// PoolSize: viper.GetInt("redis.pool_size"),
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*24)

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		zap.L().Error("redis init error, error", zap.Error(err))
		return
	}
	return
}

func Close() {
	_ = rdb.Close()
}
