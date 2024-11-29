package model

import (
	"bytes"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

type Socket interface {
	read()
	write()
	Close()
	Send(msg []byte) error
	GetUserID() string
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type WebSocketClient struct {
	conn       *websocket.Conn
	send       chan []byte
	UserID     string
	hubChannel *HubChannel
	IsClose    bool
}

func NewWebsocketClient(conn *websocket.Conn, hubChannel *HubChannel) Socket {
	client := &WebSocketClient{
		conn:       conn,
		send:       make(chan []byte, 256),
		hubChannel: hubChannel,
	}

	go client.write()
	go client.read()

	return client
}

// TODO read功能該做什麼還未確定
func (w *WebSocketClient) read() {
	defer func() {
		w.hubChannel.Unregister <- w
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
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		w.hubChannel.Broadcast <- message
	}
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

func (w *WebSocketClient) Send(msg []byte) error {
	if !w.IsClose {
		w.send <- msg
	}

	return nil
}

func (w *WebSocketClient) Close() {
	close(w.send)
	w.IsClose = true
}

func (w *WebSocketClient) GetUserID() string {
	return w.UserID
}
