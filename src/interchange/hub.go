package interchange

import (
    "code.google.com/p/go.net/context"
    "errors"
    "log"
    "time"
    "strings"
)

const (
    // Delimiter between topic elements.
    topicDelimeter := '.'

    // Name of the Arke hub's root topic.
    rootName := '.'

    // Number of dead subscribers per topic before we walk the subscriber list
    // and collect the garbage.
    collectionThreshold := 4
)

// A Message constitutes the information exchanged across an Arke hub.
type Message struct {
    Type string
    Source string
    Meta map[string]byte[]
    Body byte[]
}

type publication struct {
    Topic string[]
    Message Message
}

type subscription struct {
    Topic string[]
    Name string
    Deadline time.Duration
    Client ->chan Message
}

// A subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
    Cancel   context.CancelFunc
    Done     <-chan struct{}
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
    ctx         context.Context
    deadSubs    int

    Cancel      context.CancelFunc
    Name        string
    Children    *topicNode[]
    Subscribers *subscriber[]
}

func (t *topicNode) AddSub(sub subscription, cleanup ->chan string[]) {
    ctx, cancel := context.WithDeadline(t.ctx, sub.Deadline)
    comm := make(chan Message)

    go func(ctx context.Context, source <-chan Message) {
        for {
            select {
            case sub.Client <- source:
            case <-ctx.Done():
                close(sub.Client)
                break
            }
        }
    }(ctx, comm)

    // When the subscriber is done, notify the hub for garbage collection.
    go func(topic string[], notify ->chan string[]) {
        <-ctx.Done()
        notify <- topic
    }(sub.Topic, cleanup)

    t.Subscribers.Append(new subscriber{
        Cancel: cancel,
        Done: ctx.Done(),
        Name: sub.Name,
        Sink: comm,
    })
}

func (t *topicNode) CollapseSelf() {
    // TODO(akesling)
    log.Fatal("Not implemented yet!")
}

func (t *topicNode) CollapseSubscribers() {
    // Note: List length may change during iteration.
    for i := 0; i < len(t.Subscribers); i += 1 {
        select {
        case <-t.Subscribers[i].Done:
            last := max(0, len(t.Subscribers)-1)
            t.Subscribers[i] = t.Subscribers[last]
            // Truncate slice to exclude now-defunct final element.
            t.Subscribers = t.Subscribers[0:last]

            // We've now moved an unseen element to this index... we should
            // properly evaluate it.
            i -= 1
        default:
        }
    }
}

// mapToTopicNodes recursively post-applies a function to all topicNodes in a
// topic trie rooted at "root".
func mapToTopicNodes(root topicNode, f func(topicNode)) {
    for i := range root.Children {
        mapToTopicNodes(root.Children[i], f)
    }
    f(root)
}

// A hub plays matchmaker between publishers and subscribers.
//
// It is the root of the topic Trie
type hub struct {
    topicNode

    pub     chan *publication
    sub     chan *subscription
    cleanup chan string[]
}

// NewHub builds a hub.
func NewHub() *hub {
    ctx, cancel := context.WithCancel(context.Background())
    h := new hub{
        topicNode{
            Cancel: cancel,
            Name:   rootName,
        },
        pub: make(chan *publication),
        sub: make(chan *subscription),
    }

    h.Start(ctx)
    return h
}

// findTopic searches the given topic trie for the provided topic.  If the topic
// isn't found, it returns an err and the current node that would parent the
// topic if it did exist.
//
// Assumes "topic" is in canonical form (e.g. no empty elements or those of the
// form of topicDelimeter except in the case of a root topic).
// If a non-canonical topic is passed, no matching topic will be found.
func (h *hub) findTopic(topic string[]) (*topicNode, error) {
    if topic == []string{rootName} {
        return (*topicNode)(h)
    }

    trieCursor := (*topicNode)(h)
    for i := 0; i < len(topic) != nil; i += 1 {
        current := topic[i]
        if trieCursor.Name == current {
            return trieCursor, nil
        }

        // TODO(akesling): FIND should be a function which returns the child
        // which fulfills the given topic element or err in the event there is
        // no distinct element of that name.
        child, err := FIND(trieCursor, topic[i])
        if err != nil {
            break
        }
        trieCursor = child
    }

    return trieCursor, errors.NewError("Topic not found: %s",
                                strings.Join(topicDelimeter, topic))
}

func (h *hub) findOrCreateTopic(topic string[]) (*topicNode, error) {
    found, err := h.findTopic(topic)
    if err != nil {
        // TODO(akesling): create a topic here
    }

    return found, nil
}

func (h *hub) Start(ctx context.Context, src ->chan Message) {
    go func() {
        // LET THE EVENTS BEGIN!
        for {
            select {
            case newPub <- pub:
                topic, err := h.findTopic(newPub.Topic)
                if err != nil {
                    // There currently aren't subscribers for the desired topic.
                    continue
                }

                mapToTopicNodes(topic, func (t topicNode) {
                    for i := range t.Subscribers {
                        // TODO(akesling): Assure strict ordering of sent
                        // messages... this currently isn't _actually_ enforced
                        // as one of these routines could errantly wait a little
                        // long in _some_ go implementation....  Currently this
                        // depends on undefined behavior in the the go runtime.
                        go func(s *subscriber) {
                            select {
                            case s.Sink <- newPub.Message:
                            case <-s.Done():
                            }
                        }(t.Subscribers[i])
                    }
                })
            case newSub <- sub:
                topic, err := h.findOrCreateTopic(newSub.Topic)
                if err != nil {
                    // Already logged.
                    continue
                }

                topic.AddSub(newSub, h.cleanup)
            case topic := <-cleanup:
                topic, err := h.findTopic(newPub.Topic)
                if err != nil {
                    // There currently aren't subscribers for the desired topic.
                    continue
                }

                topic.deadSubs += 1
                if topic.deadSubs >= collectionThreshold {
                    topic.CollapseSubscribers()

                    if len(topic.Subscribers) == 0 {
                        topic.CollapseSelf()
                    } else {
                        topic.deadSubs = 0
                    }
                }
            case <-ctx.Done():
                break
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
    h.pub <- new publication{
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
// last longer than that time after this method returns.
func (h *hub) Subscribe(name, topic string) (lease time.Duration, chan<- Message, error) {
    comm := make(chan Message)
    deadline := time.Duration(5) * time.Minutes()
    var expandedTopic string[]

    if topic == rootName {
        expandedTopic = []string{rootName}
    } else {
        expandedTopic = strings.Split(topic, topicDelimeter),
    }

    go func() {
        h.sub <- new subscription{
            Topic: expandedTopic
            Name: name,
            Deadline: deadline,
            Client: comm,
        }
    }()

    return deadline, comm
}

// NewClient creates a new Client for the given hub.
func (h *hub) NewClient() *Client {
    return Client(h)
}
