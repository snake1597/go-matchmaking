package service

import (
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"go-matchmaking/model"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type HubService interface {
	Register(usedID string, conn *websocket.Conn)
}

// TODO 要有socket連線的上限
type WebSocketHubService struct {
	// TODO to sync map
	clientMap  map[string]model.Socket
	hubChannel *model.HubChannel
	broadcast  chan []byte
	nc         *nats.Conn
	cache      *redis.Client
	mq         model.MessageQueueHandler
}

func NewWebSocketHubService(
	nc *nats.Conn,
	cache *redis.Client,
	mq model.MessageQueueHandler,
) HubService {
	hub := &WebSocketHubService{
		clientMap: make(map[string]model.Socket),
		hubChannel: &model.HubChannel{
			Unregister: make(chan string),
		},
		broadcast: make(chan []byte),
		nc:        nc,
		cache:     cache,
		mq:        mq,
	}

	go hub.run()
	go hub.subscribe()

	return hub
}

func (w *WebSocketHubService) run() {
	for userID := range w.hubChannel.Unregister {
		if client, ok := w.clientMap[userID]; ok {
			delete(w.clientMap, userID)
			client.Close()
		}
	}
	{
		// 先註解 改成直接用broadcastHandle
		// for {
		// 	select {
		// 	case userID := <-w.hubChannel.Unregister:
		// 		if client, ok := w.clientMap[userID]; ok {
		// 			delete(w.clientMap, userID)
		// 			client.Close()
		// 		}
		// 	case message, ok := <-w.broadcast:
		// 		if ok {
		// 			err := w.broadcastHandle(message)
		// 			if err != nil {
		// 				log.Errorf("broadcastHandle error: %v", err)
		// 			}
		// 		}
		// 	}
		// }
	}
}

func (w *WebSocketHubService) subscribe() {
	// TODO rename public channel
	channel := "channel"
	err := w.mq.Subscribe(channel, func(message []byte) error {
		broadcastInfo := &model.BroadcastInfo{}
		err := json.Unmarshal(message, &broadcastInfo)
		if err != nil {
			return fmt.Errorf("unmarshal BroadcastInfo error: %v", err)
		}

		switch broadcastInfo.Action {
		case enum.BroadcastActionPublic:
			for userID, client := range w.clientMap {
				// TODO 暫時不用全頻廣播
				ok := client.Send([]byte{})
				if !ok {
					w.clientMap[userID].Close()
					delete(w.clientMap, userID)
				}
			}
		case enum.BroadcastActionJoin:
			m := &model.BroadcastClientInfo{
				Action: broadcastInfo.Action,
				RoomID: broadcastInfo.RoomID,
			}

			j, err := json.Marshal(m)
			if err != nil {
				return fmt.Errorf("marshal BroadcastClientInfo error: %v", err)
			}

			for _, userID := range broadcastInfo.UserIDList {
				if client, ok := w.clientMap[userID]; ok {
					client.Send(j)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Errorf("Subscribe error: %v", err)
	}
}

func (w *WebSocketHubService) Register(usedID string, conn *websocket.Conn) {
	w.clientMap[usedID] = model.NewWebsocketClient(conn, w.hubChannel, w.cache)
}
