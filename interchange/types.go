package interchange

import (
	"github.com/akesling/arke/codex"
	"golang.org/x/net/context"
	"time"
)

// Message constitutes the information exchanged across an Arke hub.
type Message struct {
	Encoding codex.Codex
	Source   string
	// Meta   map[string]string
	Body interface{}
}

// publication purveys information for routing a given message.
type publication struct {
	Topic   []string
	Message Message
}

// subscription purveys information for assembling a valid subscriber.
type subscription struct {
	Topic    []string
	Name     string
	Deadline time.Time
	Client   chan<- Message
}

// subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
	ctx  context.Context
	Name string
	sink chan<- Message
}

// createSubscriber constructs a subscriber handle from a subscription and
// context.
func createSubscriber(sub *subscription, ctx context.Context) *subscriber {
	comm := make(chan Message, subscriberBufferSize)

	new_subscriber := &subscriber{
		ctx:  ctx,
		Name: sub.Name,
		sink: comm,
	}

	go func(ctx context.Context, source <-chan Message) {
	event_loop:
		for {
			select {
			case message := <-source:
				sub.Client <- message
			case <-new_subscriber.Done():
				close(sub.Client)
				break event_loop
			}
		}
	}(ctx, comm)

	return new_subscriber
}

// Done returns a channel indicating whether the given subscriber has lost its
// lease.
func (s *subscriber) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Send handles hygienic passing of messages to a subscriber.
func (s *subscriber) Send(message *Message) {
	select {
	case s.sink <- *message:
	case <-s.Done():
	}
}
