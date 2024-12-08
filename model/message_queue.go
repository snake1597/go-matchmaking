package model

type MessageQueue interface {
	Publish(chanel string, msg []byte) error
	Subscribe(chanel string, fc func(msg []byte) error) error
	GroupSubscribe(chanel string, groupName string, fc func(msg []byte) error) error
}
