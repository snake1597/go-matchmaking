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
	cache   *redis.Client
	mq      model.MessageQueue
}

func NewNatsQueueService(
	channel string,
	nc *nats.Conn,
	mq model.MessageQueue,
) model.QueueService {
	queue := &QueueService{
		channel: channel,
		mq:      mq,
	}

	queue.Subscribe()

	return queue
}

func (q *QueueService) Publish(msg []byte) (err error) {
	err = q.mq.Publish(q.channel, msg)
	if err != nil {
		return err
	}

	return nil
}

func (q *QueueService) Subscribe() {
	go func() {
		// TODO to env, group queue name: q1
		err := q.mq.GroupSubscribe(q.channel, "q1", func(msg []byte) error {
			queueingInfo := &model.UserQueueingInfo{}
			err := json.Unmarshal(msg, &queueingInfo)
			if err != nil {
				return fmt.Errorf("unmarshal UserQueueingInfo error: %v", err)
			}

			err = q.matchmaking(queueingInfo)
			if err != nil {
				return fmt.Errorf("matchmaking error: %v, info: %+v", err, queueingInfo)
			}

			return nil
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
// ans: 改為排到redis list最前面
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
	err = n.mq.Publish("channel", info)
	if err != nil {
		return fmt.Errorf("publish error: %v", err)
	}

	return nil
}
