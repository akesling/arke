package interchange

import (
	"time"
)

type Publisher interface {
	// Produce a message to the provided topic on the Arke hub.
	Publish(topic string, message Message) error
}

type Subscriber interface {
	// Subscribe to a given topic.
	Subscribe(name, topic string, lease time.Duration) (<-chan Message, error)
}

// Client allows publication and subscription to an Arke hub.
type Client interface {
	Publisher
	Subscriber
}
