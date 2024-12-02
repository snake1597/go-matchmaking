package model

import "go-matchmaking/enum"

type BroadcastInfo struct {
	Action enum.BroadcastAction
	UserID string
}
