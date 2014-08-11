package interchange

// A Message constitutes the information exchanged across an Arke hub.
type Message struct {
    Type string
    Source string
    Meta map[string]byte[]
    Body byte[]
}

// A subscriber is a handle to an entity subscribing on an Arke hub.
type subscriber struct {
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
    Name     string
    Children *topicNode[]
    subs     subscriber[]
    end      chan bool
    Comm     chan Message
}

func (t *topicNode) Start() {
    // This must be set to 1 or destruct behaves incorrectly
    t.end = make(chan bool, 1)

    go func() {
        for {
            select {
            case signal := <-t.end:
                break
            case  message := <-Comm:
                for i := subs {
                    // TODO(akesling): make it safe to spin this off as a
                    // go routine.  Currently destruction can race with an
                    // asynchronous send.
                    t.subs[i].Sink <- message
                }

                for i := t.children {
                    t.children[i].Comm <- message
                }
            }
        }
        // Apparently we're being shut down.

        // Let's shut down our children.
        for i := t.children {
            t.children[i].destruct()
        }

        // And close our subscribers.
        for i := t.subs {
            close(t.subs[i].Sink)
        }
    }()
}

func (t *topicNode) destruct() {
    select {
    case t.end <- true:
    default:
        log.Error('Multiple call to topicNode.destruct().')
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

func (h *hub) Subscribe(topic string) (secondsLease int, chan<- Message, error) {
}

// NewClient creates a new client for the given hub.
func (h *hub) NewClient() *Client {
    return Client(h)
}
