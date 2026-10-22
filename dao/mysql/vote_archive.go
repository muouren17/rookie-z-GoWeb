package mysql

import (
	"context"
	"webapp/models"
)

func SavetimeoutPost(ctx context.Context, postdatas *[]models.ParamSaveClosedPost) (err error) {
	sqlstr := `insert into post_vote_stat 
	values(:post_id,:up_count,:down_count,:score,:archived_at) 
	ON DUPLICATE KEY UPDATE 
    up_count = :up_count,
    down_count = :down_count,
    score = :score,
    archived_at = :archived_at`

	for _, postdata := range *postdatas {
		if err = ctx.Err(); err != nil {
			return
		}
		if _, err = db.NamedExecContext(ctx, sqlstr, postdata); err != nil {
			return
		}

	}
	return

}
