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
	ctx  context.Context
	Name string
	sink chan<- Message
}

func CreateSubscriber(sub *subscription, ctx context.Context) *subscriber {
	comm := make(chan Message)

	go func(ctx context.Context, source <-chan Message) {
	event_loop:
		for {
			select {
			case message := <-source:
				sub.Client <- message
			case <-ctx.Done():
				close(sub.Client)
				break event_loop
			}
		}
	}(ctx, comm)

	return &subscriber{
		ctx:  ctx,
		Name: sub.Name,
		sink: comm,
	}
}

func (s *subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *subscriber) Send(message *Message) {
	s.sink <- *message
}
