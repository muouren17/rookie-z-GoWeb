package models

const (
	OrderTime  = "time"
	OrderScore = "score"
)

type ParamSigUp struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RePassword string `json:"repassword" binding:"required,eqfield=Password"`
}

type ParamLogin struct {
	Username string `db:"username" json:"username" binding:"required"`
	Password string `db:"password" json:"password" binding:"required"`
}

type ParamPair struct {
	UserID int64  `json:"user_id,string" binding:"required"`
	Token  string `json:"token" binding:"required"`
}

type ParamPost struct {
	AuthorID    int64  `json:"-" db:"author_id"`
	CommunityID int64  `json:"community_id" db:"community_id" binding:"required,min=1,max=4"`
	Title       string `json:"title" db:"title" binding:"required"`
	Content     string `json:"content" db:"content" binding:"required"`
}

type ParamVoteData struct {
	PostID    int64 `json:"post_id,string" binding:"required"`
	Direction int8  `json:"direction,string" binding:"oneof=1 0 -1"` //赞成1，反对-1，弃权0
}

type ParamPostList struct {
	Page        int64  `json:"page" form:"page"`
	Size        int64  `form:"size"`
	Order       string `form:"order"`
	CommunityID int64  `json:"community_id" form:"community_id"`
}
