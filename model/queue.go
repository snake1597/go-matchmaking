package model

type QueueHandler interface {
	Publish(msg []byte) error
	Subscribe()
}
