package model

import (
	"context"
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type Socket interface {
	read()
	write()
	Close()
	Send(msg []byte) bool
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
)

type WebSocketClient struct {
	send       chan []byte
	UserID     string
	hubChannel *HubChannel
	IsClose    bool
	conn       *websocket.Conn
	cache      *redis.Client
}

func NewWebsocketClient(
	conn *websocket.Conn,
	hubChannel *HubChannel,
	cache *redis.Client,
) Socket {
	client := &WebSocketClient{
		conn:       conn,
		send:       make(chan []byte, 256),
		hubChannel: hubChannel,
		cache:      cache,
	}

	go client.write()
	go client.read()

	return client
}

func (w *WebSocketClient) read() {
	defer func() {
		w.hubChannel.Unregister <- w.UserID
		w.Close()
	}()

	w.conn.SetReadLimit(maxMessageSize)
	w.conn.SetReadDeadline(time.Now().Add(pongWait))
	w.conn.SetPongHandler(func(string) error { w.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		err = w.messageHandle(message)
		if err != nil {
			log.Errorf("messageHandle error: %v", err)
		}
	}
}

func (w *WebSocketClient) messageHandle(message []byte) (err error) {
	broadcastClientInfo := &BroadcastClientInfo{}
	err = json.Unmarshal(message, &broadcastClientInfo)
	if err != nil {
		return fmt.Errorf("unmarshal BroadcastInfo error: %v", err)
	}

	switch broadcastClientInfo.Action {
	case enum.BroadcastActionJoin:
		room, err := w.cache.HGetAll(context.Background(), fmt.Sprintf(enum.RoomKey, broadcastClientInfo.RoomID)).Result()
		if err != nil {
			return fmt.Errorf("HGetAll error: %v", err)
		}

		if len(room) == 0 {
			// TODO hub publish to the nats
		}

		confirmCount := 0
		for k, v := range room {
			if k == w.UserID {
				v = "confirm"
			}

			if v == "confirm" {
				confirmCount += 1
			}
		}

		if confirmCount != 10 {
			return nil
		}

		// TODO 通知使用者可開始遊戲
	}

	return nil
}

func (w *WebSocketClient) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		w.Close()
	}()

	for {
		select {
		case message, ok := <-w.send:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				w.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			writer, err := w.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			writer.Write(message)

			n := len(w.send)
			for i := 0; i < n; i++ {
				writer.Write(newline)
				writer.Write(<-w.send)
			}

			if err := writer.Close(); err != nil {
				return
			}
		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (w *WebSocketClient) Send(msg []byte) bool {
	if w.IsClose {
		return false
	}
	w.send <- msg
	return true
}

func (w *WebSocketClient) Close() {
	close(w.send)
	w.IsClose = true
}
