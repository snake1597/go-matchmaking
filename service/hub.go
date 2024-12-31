package service

import (
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"go-matchmaking/model"
	"sync"

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
	clientMap  sync.Map
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
		clientMap: sync.Map{},
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
		if client, ok := w.clientMap.Load(userID); ok {
			w.clientMap.Delete(userID)
			socket, ok := client.(model.Socket)
			if ok {
				socket.Close()
			}
		}
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
			w.clientMap.Range(func(key any, value any) bool {
				socket, ok := value.(model.Socket)
				if ok {
					// TODO 暫時不用全頻廣播
					isSend := socket.Send([]byte{})
					if !isSend {
						w.clientMap.Delete(key)
						socket.Close()
					}
				}

				return true
			})
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
				if client, ok := w.clientMap.Load(userID); ok {
					socket, ok := client.(model.Socket)
					if ok {
						socket.Send(j)
					}
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
	socket := model.NewWebsocketClient(conn, w.hubChannel, w.cache)
	w.clientMap.Store(usedID, socket)
}
