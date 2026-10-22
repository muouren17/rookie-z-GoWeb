package jobs

import (
	"context"
	"time"
	"webapp/logic"

	"go.uber.org/zap"
)

// func Vote_archive() {
// 	forever := make(chan int)
// 	go func() {
// 		for true {
// 			time.Sleep(time.Minute * 30)
// 			err := logic.Vote_archive()
// 			if err != nil {
// 				zap.L().Error("回收过时帖子失败", zap.Error(err))
// 				return
// 			}
// 		}
// 	}()
// 	<-forever
// }

func Vote_archive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			roundctx, cancel := context.WithTimeout(ctx, time.Minute)
			err := logic.Vote_archive(roundctx)
			cancel()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				zap.L().Error("投票归档失败", zap.Error(err))
			}
		}
	}

}
