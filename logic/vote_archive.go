package logic

import (
	"context"
	"strconv"
	"time"
	"webapp/dao/mysql"
	"webapp/dao/redis"
	"webapp/models"
)

func Vote_archive(ctx context.Context) error {
	post_timeout_ids, err := redis.GettimeoutPostID(ctx)
	if err != nil {
		return err
	}
	err = redis.SavetimeoutPostID(ctx, post_timeout_ids)
	if err != nil {
		return err
	}
	upcount, err := redis.GetPostVoteData(ctx, post_timeout_ids)
	if err != nil {
		return err
	}
	downcount, err := redis.GetPostDownData(ctx, post_timeout_ids)
	if err != nil {
		return err
	}

	archivetime := time.Now()
	var postdatas []models.ParamSaveClosedPost

	for i := 0; i < len(post_timeout_ids); i++ {
		score, err := redis.GetPostScore(ctx, post_timeout_ids[i])
		if err != nil {
			return err
		}
		temp := new(models.ParamSaveClosedPost)
		postID, err := strconv.Atoi(post_timeout_ids[i])
		if err != nil {
			return err
		}
		temp.PostID = int64(postID)
		temp.UpCount = upcount[i]
		temp.DownCount = downcount[i]
		temp.Score = score
		temp.ArchiveAt = archivetime
		postdatas = append(postdatas, *temp)

	}

	err = mysql.SavetimeoutPost(ctx, &postdatas)
	if err != nil {
		return err
	}

	err = redis.RemtimeoutPost(ctx, post_timeout_ids)
	if err != nil {
		return err
	}

	return nil
}
