package nats

import (
	"fmt"
	"go-matchmaking/model"

	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
)

type Client struct {
	nc *nats.Conn
}

func NewClient(
	token string,
	address string,
	port int,
) model.MessageQueue {
	s := fmt.Sprintf("%s@%s:%d", token, address, port)
	nc, err := nats.Connect(s)
	if err != nil {
		log.Fatal(err)
	}

	return &Client{
		nc: nc,
	}
}

func (c *Client) Publish() {}

func (c *Client) Subscribe(chanel string, fc func(msg []byte) error) error {
	_, err := c.nc.Subscribe(chanel, func(m *nats.Msg) {
		err := fc(m.Data)
		if err != nil {
			log.Errorf("receive msg error: %v", err)
		}
	})
	if err != nil {
		return err
	}

	return nil
}
