package service

import (
	"go-matchmaking/model"

	"github.com/gorilla/websocket"
)

type HubService interface {
	Register(usedID string, conn *websocket.Conn)
	Broadcast(msg []byte)
}

type WebSocketHubService struct {
	clientMap  map[string]model.Socket
	hubChannel *model.HubChannel
}

func NewWebSocketHubService() HubService {
	hub := &WebSocketHubService{
		clientMap: make(map[string]model.Socket),
		hubChannel: &model.HubChannel{
			Broadcast:  make(chan []byte),
			Unregister: make(chan model.Socket),
		},
	}

	go hub.run()

	return hub
}

func (w *WebSocketHubService) run() {
	for {
		select {
		case client := <-w.hubChannel.Unregister:
			if _, ok := w.clientMap[client.GetUserID()]; ok {
				delete(w.clientMap, client.GetUserID())
				client.Close()
			}
		case message := <-w.hubChannel.Broadcast:
			for userID := range w.clientMap {
				err := w.clientMap[userID].Send(message)
				if err != nil {
					w.clientMap[userID].Close()
					delete(w.clientMap, userID)
				}
			}
		}
	}
}

func (w *WebSocketHubService) Register(usedID string, conn *websocket.Conn) {
	w.clientMap[usedID] = model.NewWebsocketClient(conn, w.hubChannel)
}

func (w *WebSocketHubService) Broadcast(msg []byte) {
	w.hubChannel.Broadcast <- msg
}
