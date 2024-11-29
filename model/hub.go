package model

type HubChannel struct {
	Broadcast  chan []byte
	Unregister chan Socket
}
