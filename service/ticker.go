package service

import (
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type TickerService struct {
	nc         *nats.Conn
	jobChannel chan string
}

func NewTickerService(
	nc *nats.Conn,
) *TickerService {
	srv := &TickerService{
		nc:         nc,
		jobChannel: make(chan string, 10), // TODO worker pool size to env
	}

	srv.subscribe()
	srv.buildWorkerPool()

	return srv
}

func (t *TickerService) subscribe() {
	go func() {
		// TODO to env
		_, err := t.nc.Subscribe("ticker", func(m *nats.Msg) {
			t.jobChannel <- string(m.Data)
			log.Printf("Received a message from %s : %s\n", "ticker", string(m.Data))
		})
		if err != nil {
			log.Errorf("nats subscribe error: %v", err)
		}
	}()
}

func (t *TickerService) buildWorkerPool() {
	for i := 0; i < 10; i++ {
		go t.worker(t.jobChannel)
	}
}

func (t *TickerService) worker(ch chan string) {
	for roomID := range ch {
		// TODO
		// find roomID at redis
		// if found it
		// publish msg to client
		// counting 10 sec
		log.Println(roomID)
	}
}
