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
