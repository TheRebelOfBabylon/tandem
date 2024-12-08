package websocket

import "github.com/TheRebelOfBabylon/tandem/msg"

type ConnectionHandler interface {
	Start() error
	Stop() error
	SendChannel() chan msg.Msg
}
