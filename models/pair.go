package models

type Pair struct {
	UserID int64  `json:"user_id,string" `
	Token  string `json:"token" `
}
