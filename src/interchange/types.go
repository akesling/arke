package interchange

import (
	"code.google.com/p/go.net/context"
	"time"
)

// A Message constitutes the information exchanged across an Arke hub.
type Message struct {
	Type   string
	Source string
	Meta   map[string][]byte
	Body   []byte
}

type publication struct {
	Topic   []string
	Message Message
}

type subscription struct {
	Topic    []string
	Name     string
	Deadline time.Time
	Client   chan<- Message
}

// A subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
	ctx    context.Context
	Cancel context.CancelFunc
	Done   <-chan struct{}
	Name   string
	Sink   chan<- Message
}

func (s *subscriber) Send(message *Message) {
	// TODO(akesling): Assure strict ordering of sent
	// messages... this currently isn't _actually_ enforced
	// as one of these routines could errantly wait a little
	// long in _some_ go implementation....  Currently this
	// depends on undefined behavior in the go runtime.
	go func(sub *subscriber) {
		select {
		case sub.Sink <- *message:
		case <-sub.Done:
		}
	}(s)
}
