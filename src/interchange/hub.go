package interchange

import (
    "code.google.com/p/go.net/context"
    "errors"
    "fmt"
    "log"
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
    Cancel   context.CancelFunc
    Done     <-chan struct{}
    Name     string
    Sink     chan<- Message
}

// A topicNode represents a vertex in the topic trie.
//
// Unlike a standard trie, the topic trie is fully expanded on each token.
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
    Children    []*topicNode
    Subscribers []*subscriber
}

func (t *topicNode) AddSub(sub *subscription, cleanup chan<- []string) {
    ctx, cancel := context.WithDeadline(t.ctx, sub.Deadline)
    comm := make(chan Message)

    go func(ctx context.Context, source <-chan Message) {
        for {
            select {
            case message := <- source:
                sub.Client <- message
            case <-ctx.Done():
                close(sub.Client)
                break
            }
        }
    }(ctx, comm)

    // When the subscriber is done, notify the hub for garbage collection.
    go func(topic []string, notify chan<- []string) {
        <-ctx.Done()
        notify <- topic
    }(sub.Topic, cleanup)

    t.Subscribers = append(t.Subscribers, &subscriber{
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

func (t *topicNode) CreateChild(subTopic []string) (newTopic *topicNode, err error) {
    // TODO(akesling)
    log.Fatal("Not implemented yet!")
    return nil, nil
}

func (t *topicNode) CollapseSubscribers() {
    // Note: List length may change during iteration.
    for i := 0; i < len(t.Subscribers); i += 1 {
        select {
        case <-t.Subscribers[i].Done:
            var last uint
            if len(t.Subscribers) > 0 {
                last = uint(len(t.Subscribers)-1)
            }

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
func mapToTopicNodes(root *topicNode, f func(*topicNode)) {
    for i := range root.Children {
        mapToTopicNodes(root.Children[i], f)
    }
    f(root)
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
    h := &hub{
        root: &topicNode{
            Cancel: cancel,
            Name:   rootName,
        },
        pub: make(chan *publication),
        sub: make(chan *subscription),
    }

    h.Start(ctx)
    return h
}

func binarySearch(needle string, haystack []*topicNode) (index int) {
    log.Fatal("Not implemented yet!")
    return -1
}

func findTopic(needle string, haystack []*topicNode) (*topicNode, error) {
    index := binarySearch(needle, haystack)

    if index < 0 {
        return nil, errors.New("Topic node not found.")
    }

    return haystack[index], nil
}

// maybeFindTopic searches the given topic trie for the provided topic.  If the
// topic isn't found, it returns an err and the current node that would parent
// the topic if it did exist.
//
// Assumes "topic" is in canonical form (e.g. no empty elements or those of the
// form of topicDelimeter except in the case of a root topic).
// If a non-canonical topic is passed, no matching topic will be found.
func (h *hub) maybeFindTopic(topic []string) (localRoot *topicNode, rest []string) {
    if len(topic) == 1 && topic[0]  == "." {
        return h.root, []string{}
    }

    localRoot = h.root
    rest = topic[:]
    for i := 0; i < len(topic); i += 1 {
        current := topic[i]
        if localRoot.Name == current {
            return localRoot, nil
        }

        child, err := findTopic(topic[i], localRoot.Children)
        if err != nil {
            break
        }
        localRoot = child
        rest = topic[i:]
    }

    return localRoot, rest
}

func (h *hub) findTopic(topic []string) (*topicNode, error) {
    found, rest := h.maybeFindTopic(topic)
    if len(rest) != 0 {
        return nil, errors.New(
            fmt.Sprintf("Topic not found: %s",
                        strings.Join(topic, topicDelimeter)))
    }
    return found, nil
}

func (h *hub) findOrCreateTopic(topic []string) (*topicNode, error) {
    found, rest := h.maybeFindTopic(topic)

    var err error
    if len(rest) != 0 {
        found, err = found.CreateChild(rest)
    }

    return found, err
}

func (h *hub) Start(ctx context.Context) {
    go func() {
        // LET THE EVENTS BEGIN!
        for {
            select {
            case newPub := <-h.pub:
                topic, err := h.findTopic(newPub.Topic)
                if err != nil {
                    // There currently aren't subscribers for the desired topic.
                    continue
                }

                mapToTopicNodes(topic, func (t *topicNode) {
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
                        tNode.CollapseSelf()
                    } else {
                        tNode.deadSubs = 0
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
