package model

type MessageQueue interface {
	Publish()
	Subscribe(chanel string, fc func(msg []byte) error) error
}
