package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-matchmaking/enum"
	"go-matchmaking/model"
	"time"

	"go-matchmaking/pkg/lua"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type QueueService struct {
	channel string
	nc      *nats.Conn
	cache   *redis.Client
	mq      model.MessageQueue
}

func NewNatsQueueService(
	channel string,
	nc *nats.Conn,
) model.QueueService {
	queue := &QueueService{
		channel: channel,
		nc:      nc,
	}

	queue.subscribe()

	return queue
}

func (n *QueueService) Publish(msg []byte) (err error) {
	err = n.nc.Publish(n.channel, msg)
	if err != nil {
		return err
	}

	return nil
}

func (n *QueueService) subscribe() {
	go func() {
		// TODO to env, group queue name: q1
		_, err := n.nc.QueueSubscribe(n.channel, "q1", func(m *nats.Msg) {
			queueingInfo := &model.UserQueueingInfo{}
			err := json.Unmarshal([]byte(m.Data), &queueingInfo)
			if err != nil {
				log.Errorf("Unmarshal UserQueueingInfo error: %v", err)
			}

			err = n.matchmaking(queueingInfo)
			if err != nil {
				log.Errorf("matchmaking error: %v, info: %+v", err, queueingInfo)
			}

			log.Printf("Received a message from %s : %s\n", n.channel, string(m.Data))
		})
		if err != nil {
			log.Errorf("nats subscribe error: %v", err)
		}
	}()
}

// TODO 延伸
// 配對後能取消
// 根據rank來匹配

// 問題
// 要如何確認client可以被加入房間
// 能確認是否加入 需要有時間限制 時間一到 原確認的要續留 再補上空缺
func (n *QueueService) matchmaking(queueingInfo *model.UserQueueingInfo) (err error) {
	ctx := context.Background()

	cacheKeyList := []string{
		"room",
	}

	roomSize := 10
	args := []interface{}{
		roomSize, // TODO 房間人數, to env
		queueingInfo.UserID,
	}

	userIDList, err := lua.JoinRoom().Run(ctx, n.cache, cacheKeyList, args).StringSlice()
	if err != nil {
		return fmt.Errorf("JoinRoom error: %v", err)
	}

	if len(userIDList) < roomSize {
		return nil
	}

	broadcastInfo := model.BroadcastInfo{
		Action:     enum.BroadcastActionJoin,
		UserIDList: userIDList,
		RoomID:     uuid.NewString(),
	}

	setValue := make(map[string]string)
	for _, userID := range broadcastInfo.UserIDList {
		setValue[userID] = ""
	}

	// TODO cache key to enum
	err = n.cache.HSet(ctx, fmt.Sprintf("room_%s", broadcastInfo.RoomID), setValue).Err()
	if err != nil {
		return fmt.Errorf("HSet error: %v", err)
	}

	// TODO 跟HSet一起包成lua
	err = n.cache.Expire(ctx, fmt.Sprintf("room_%s", broadcastInfo.RoomID), 10*time.Second).Err()
	if err != nil {
		return fmt.Errorf("HSet error: %v", err)
	}

	info, err := json.Marshal(broadcastInfo)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}

	// TODO rename public channel
	err = n.nc.Publish("channel", info)
	if err != nil {
		return fmt.Errorf("publish error: %v", err)
	}

	return nil
}
