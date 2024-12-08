package model

type QueueService interface {
	Publish(msg []byte) error
	Subscribe()
}
