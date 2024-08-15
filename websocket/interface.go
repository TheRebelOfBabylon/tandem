package websocket

import "github.com/TheRebelOfBabylon/tandem/msg"

type ConnectionHandler interface {
	Start() error
	Error() error
	SendChannel() chan msg.Msg
}
