package interchange

import (
    "code.google.com/p/go.net/context"
)

// A Message constitutes the information exchanged across an Arke hub.
type Message struct {
    Type string
    Source string
    Meta map[string]byte[]
    Body byte[]
}

// A subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
    Cancel   context.CancelFunc
    Name     string
    Sink     ->chan Message
}

// A topicNode represents a vertex in the topic trie.
//
// Topic nodes occur at the interface between topic segments.  E.g. a topic node
// will exist for each partition in the topic string foo.bar.baz plus one
// implicit node at the root.  The root's name is implicitly '.'.
// Further, for the hub consisting of subscribers on foo.bar.baz and foo.bar.qux
// there will be five topic nodes: one for the root and each of foo and bar...
// and one for each of baz and qux.
type topicNode struct {
    Cancel   context.CancelFunc
    Name     string
    Children *topicNode[]
    subs     subscriber[]
    Comm     chan Message
}

func (t *topicNode) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                break
            case  message := <-Comm:
                for i := range subs {
                    go func(s subscriber) {
                        select {
                        case s.Sink <- message:
                        case <-s.Done():
                            // We're done, no need to send.
                        }
                    }(t.subs[i])
                }

                for i := range t.children {
                    t.children[i].Comm <- message
                }
            }
        }
        // Apparently we're being shut down.

        // Let's shut down our children.
        for i := t.children {
            t.children[i].Cancel()
        }

        // And close our subscribers.
        for i := t.subs {
            t.subs[i].Cancel()
        }
    }()
}

type Destructor interface {
    Done chan bool
    func Destruct()
}

func (d *Destructor) Destruct() {
    select {
    case d.Done <- true:
    default:
        log.Error('Multiple call to destruct().')
    }
}

// A hub plays matchmaker between publishers and subscribers.
//
// It is the root of topic Trie
type hub topicNode

// NewHub builds a hub.
func NewHub() *hub {
    h := *hub(new topicNode{'.'})
    h.Start()
    return interchange
}

func (h *hub) Publish(topic string, message Message) error {
    go func() {
        h.Comm <- message
    }()
}

func (h *hub) Subscribe(name, topic string) (lease time.Duration, chan<- Message, error) {
    comm := make(chan Message)
    sub := subscriber{

    }

    h.addSubscriberToTopic(topic, sub)
    return (time.Duration(5) * time.Minutes()), comm

}

// NewClient creates a new client for the given hub.
func (h *hub) NewClient() *Client {
    return Client(h)
}
