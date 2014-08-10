package interchange

type Publisher interface {
    // Produce a message to the provided topic on the Arke hub.
    Publish(topic string, message Message) error
}

type Subscriber interface {
    // Subscribe to a given topic.
    Subscribe(topic string) (secondsLease int, chan<- Message, error)
}

// Client allows publication and subscription to/from an Arke hub.
type Client interface {
    Publisher
    Subscriber
}
