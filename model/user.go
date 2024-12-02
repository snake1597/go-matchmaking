package model

type UserQueueingInfo struct {
	UserID string `json:"user_id"`
	Rank   int64  `json:"rank"`
}
