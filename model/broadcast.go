package model

import "go-matchmaking/enum"

type BroadcastInfo struct {
	Action     enum.BroadcastAction
	UserIDList []string
	RoomID     string
}

type BroadcastClientInfo struct {
	Action enum.BroadcastAction
	RoomID string
}
