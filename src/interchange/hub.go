package interchange

import (
    "code.google.com/p/go.net/context"
    "errors"
    "fmt"
    "time"
    "strings"
)

const (
    // Delimiter between topic elements.
    topicDelimeter = "."

    // Name of the Arke hub's root topic.
    rootName = "."

    // Number of dead subscribers per topic before we walk the subscriber list
    // and collect the garbage.
    collectionThreshold = 4
)

// A Message constitutes the information exchanged across an Arke hub.
type Message struct {
    Type string
    Source string
    Meta map[string][]byte
    Body []byte
}

type publication struct {
    Topic []string
    Message Message
}

type subscription struct {
    Topic []string
    Name string
    Deadline time.Time
    Client chan<- Message
}

// A subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
    ctx      context.Context
    Cancel   context.CancelFunc
    Done     <-chan struct{}
    Name     string
    Sink     chan<- Message
}

// A hub plays matchmaker between publishers and subscribers.
//
// It contains the root of the topic Trie
//
// hub is thread-hostile, only one go-routine should access it at a time.
type hub struct {
    root    *topicNode

    pub     chan *publication
    sub     chan *subscription
    cleanup chan []string
}

// NewHub builds a hub.
func NewHub() *hub {
    ctx, cancel := context.WithCancel(context.Background())

    new_root := newTopicNode(ctx, cancel, []string{rootName})

    h := &hub{
        root: new_root,
        pub: make(chan *publication),
        sub: make(chan *subscription),
    }

    h.Start(ctx)
    return h
}

func (h *hub) findTopic(topic []string) (*topicNode, error) {
    found, rest, _ := h.root.MaybeFindTopic(topic)

    if len(rest) != 0 {
        return nil, errors.New(
            fmt.Sprintf("Topic not found: %s",
                        strings.Join(topic, topicDelimeter)))
    }

    return found, nil
}

func (h *hub) findOrCreateTopic(topic []string) (*topicNode, error) {
    found, rest, _ := h.root.MaybeFindTopic(topic)

    var err error
    if len(rest) != 0 {
        found, err = found.CreateChild(rest)
    }

    return found, err
}

func (h *hub) Start(ctx context.Context) {
    go func() {
        // LET THE EVENTS BEGIN!
        event_loop:
        for {
            select {
            case newPub := <-h.pub:
                topic, err := h.findTopic(newPub.Topic)
                if err != nil {
                    // There currently aren't subscribers for the desired topic.
                    // TODO(akesling): log INFO output.
                    continue
                }

                topic.Map(nil, func (_, t *topicNode) {
                    for i := range t.Subscribers {
                        // TODO(akesling): Assure strict ordering of sent
                        // messages... this currently isn't _actually_ enforced
                        // as one of these routines could errantly wait a little
                        // long in _some_ go implementation....  Currently this
                        // depends on undefined behavior in the the go runtime.
                        go func(s *subscriber) {
                            select {
                            case s.Sink <- newPub.Message:
                            case <-s.Done:
                            }
                        }(t.Subscribers[i])
                    }
                })
            case newSub := <-h.sub:
                topic, err := h.findOrCreateTopic(newSub.Topic)
                if err != nil {
                    // Already logged.
                    continue
                }

                topic.AddSub(newSub, h.cleanup)
            case topic := <-h.cleanup:
                tNode, err := h.findTopic(topic)
                if err != nil {
                    // There currently aren't subscribers for the desired topic.
                    continue
                }

                tNode.deadSubs += 1
                if tNode.deadSubs >= collectionThreshold {
                    tNode.CollapseSubscribers()

                    if len(tNode.Subscribers) == 0 {
                        tNode.Collapse()
                    } else {
                        tNode.deadSubs = 0
                    }
                }
            case <-ctx.Done():
                break event_loop
            }
        }
    }()
}

// Publish synchronously publishes the message on the given topic.
//
// Return does not imply actual delivery, only that the message is now queued
// for all subscribers current as of return time.
//
// The only strict guarantee for published messages is that any given subscriber
// will receive them in the order sent by a given client (e.g. publications
// from multiple clients may be interleaved, but per-client ordering is
// guaranteed).
func (h *hub) Publish(topic string, message Message) error {
    h.pub <- &publication{
        Topic: strings.Split(topic, topicDelimeter),
        Message: message,
    }

    return nil
}

// Subscribe asynchronously issues a subscription request.
//
// Return does not imply that a subscriber is active yet, only that the
// subscription is now queued.  The subscription lease returned is the minimum
// amount of time that this subscriber may be active... the subscription may
// last longer than that time after Subscribe() invocation.
func (h *hub) Subscribe(name, topic string, lease time.Duration) (chan<- Message, error) {
    deadline := time.Now().Add(lease)
    comm := make(chan Message)
    var expandedTopic []string

    if topic == rootName {
        expandedTopic = []string{rootName}
    } else {
        expandedTopic = strings.Split(topic, topicDelimeter)
    }

    go func() {
        h.sub <- &subscription{
            Topic: expandedTopic,
            Name: name,
            Deadline: deadline,
            Client: comm,
        }
    }()

    return comm, nil
}

// NewClient creates a new Client for the given hub.
func (h *hub) NewClient() *Client {
    temp := Client(h)
    return &temp
}
