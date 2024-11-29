package service

import (
	"encoding/json"
	"go-matchmaking/model"

	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
)

type QueueService interface {
	Publish(msg []byte) error
	subscribe()
}

type NatsQueueService struct {
	nc      *nats.Conn
	channel string
}

func NewNatsQueueService(
	nc *nats.Conn,
	channel string,
) QueueService {
	queue := &NatsQueueService{
		nc:      nc,
		channel: channel,
	}

	queue.subscribe()

	return queue
}

func (n *NatsQueueService) Publish(msg []byte) (err error) {
	err = n.nc.Publish(n.channel, msg)
	if err != nil {
		return err
	}

	return nil
}

func (n *NatsQueueService) subscribe() {
	go func() {
		// TODO to env, group queue name: q1
		_, err := n.nc.QueueSubscribe(n.channel, "q1", func(m *nats.Msg) {
			queueingInfo := &model.UserQueueingInfo{}
			err := json.Unmarshal([]byte(m.Data), &queueingInfo)
			if err != nil {
				log.Errorf("Unmarshal UserQueueingInfo error: %v", err)
			}

			n.matchmaking(queueingInfo)

			log.Printf("Received a message from %s : %s\n", n.channel, string(m.Data))
		})
		if err != nil {
			log.Errorf("nats subscribe error: %v", err)
		}
	}()
}

func (n *NatsQueueService) matchmaking(queueingInfo *model.UserQueueingInfo) {

}
