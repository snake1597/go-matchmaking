package service

import (
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"go-matchmaking/model"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type HubService interface {
	Register(usedID string, conn *websocket.Conn)
	Broadcast(msg []byte)
}

type WebSocketHubService struct {
	// TODO to sync map
	clientMap  map[string]model.Socket
	hubChannel *model.HubChannel
	nc         *nats.Conn
}

func NewWebSocketHubService(
	nc *nats.Conn,
) HubService {
	hub := &WebSocketHubService{
		clientMap: make(map[string]model.Socket),
		hubChannel: &model.HubChannel{
			Broadcast:  make(chan []byte),
			Unregister: make(chan string),
		},
		nc: nc,
	}

	go hub.run()
	go hub.subscribe()

	return hub
}

func (w *WebSocketHubService) run() {
	for {
		select {
		case userID := <-w.hubChannel.Unregister:
			if client, ok := w.clientMap[userID]; ok {
				delete(w.clientMap, userID)
				client.Close()
			}
		case message := <-w.hubChannel.Broadcast:
			err := w.broadcastHandle(message)
			if err != nil {
				log.Errorf("broadcastHandle error: %v", err)
			}
		}
	}
}

func (w *WebSocketHubService) subscribe() {
	// TODO rename public channel
	channel := "channel"
	_, err := w.nc.Subscribe(channel, func(m *nats.Msg) {
		w.Broadcast(m.Data)
	})
	if err != nil {
		log.Errorf("nats subscribe error: %v", err)
	}
}

func (w *WebSocketHubService) broadcastHandle(message []byte) error {
	broadcastInfo := &model.BroadcastInfo{}
	err := json.Unmarshal(message, &broadcastInfo)
	if err != nil {
		return fmt.Errorf("unmarshal BroadcastInfo error: %v", err)
	}

	switch broadcastInfo.Action {
	case enum.BroadcastActionPublic:
		for userID, client := range w.clientMap {
			ok := client.Send(message)
			if !ok {
				w.clientMap[userID].Close()
				delete(w.clientMap, userID)
			}
		}
	case enum.BroadcastActionJoin:
		if client, ok := w.clientMap[broadcastInfo.UserID]; ok {
			client.Send(message)
		}
	}

	return nil
}

func (w *WebSocketHubService) Register(usedID string, conn *websocket.Conn) {
	w.clientMap[usedID] = model.NewWebsocketClient(conn, w.hubChannel)
}

func (w *WebSocketHubService) Broadcast(msg []byte) {
	w.hubChannel.Broadcast <- msg
}
