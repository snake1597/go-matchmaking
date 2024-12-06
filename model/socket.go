package model

import (
	"context"
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"go-matchmaking/pkg/lua"
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
	broadcastInfo := &BroadcastInfo{}
	err = json.Unmarshal(message, &broadcastInfo)
	if err != nil {
		return fmt.Errorf("unmarshal BroadcastInfo error: %v", err)
	}

	switch broadcastInfo.Action {
	case enum.BroadcastActionJoin:
		cacheKeyList := []string{
			fmt.Sprintf("room_%s", broadcastInfo.RoomID),
		}

		args := []interface{}{
			broadcastInfo.UserIDList[0],
		}

		confirmCount, err := lua.ConfirmJoinRoom().Run(context.Background(), w.cache, cacheKeyList, args).Int()
		if err != nil {
			return fmt.Errorf("ConfirmJoinRoom error: %v", err)
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
