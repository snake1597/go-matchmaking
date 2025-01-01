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
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type QueueService struct {
	cache *redis.Client
	mq    model.MessageQueueHandler
}

func NewNatsQueueService(
	cache *redis.Client,
	mq model.MessageQueueHandler,
) model.QueueHandler {
	queue := &QueueService{
		cache: cache,
		mq:    mq,
	}

	queue.Subscribe()

	return queue
}

func (q *QueueService) Publish(msg []byte) (err error) {
	err = q.mq.Publish(enum.GroupChannel, msg)
	if err != nil {
		return err
	}

	return nil
}

func (q *QueueService) Subscribe() {
	go func() {
		err := q.mq.GroupSubscribe(enum.GroupChannel, enum.GroupName, func(msg []byte) error {
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

	err = n.cache.Expire(ctx, fmt.Sprintf(enum.RoomKey, broadcastInfo.RoomID), 10*time.Second).Err()
	if err != nil {
		return fmt.Errorf("HSet error: %v", err)
	}

	err = n.cache.HSet(ctx, fmt.Sprintf(enum.RoomKey, broadcastInfo.RoomID), setValue).Err()
	if err != nil {
		return fmt.Errorf("HSet error: %v", err)
	}

	info, err := json.Marshal(broadcastInfo)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}

	err = n.mq.Publish(enum.HubBroadcastChannel, info)
	if err != nil {
		return fmt.Errorf("publish error: %v", err)
	}

	return nil
}
