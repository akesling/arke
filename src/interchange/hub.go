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
    Name       string
    Children   *topicNode[]
    subs       subscriber[]
}

func (t *topicNode) Push(message Message) {
    for i := subs {
        go func() {
            subs[i].Sink <- message
        }
    }

    for i := t.children {
        go t.children[i].Push(message)
    }
}

// A hub plays matchmaker between publishers and subscribers.
type hub struct {
    topicTrie topicNode*
}

// NewHub builds a hub.
func NewHub() *hub {
    return new hub{new topicNode{'.'}}
}

func (h *hub) Publish(topic string, message Message) error {
    base, err := findAndMergeTopic(topic)
    if err != nil {
        log.Fatal(err)
    }

    go base.Push(message)
}

func (h *hub) Subscribe(topic string) (secondsLease int, chan<- Message, error) {
}

// NewClient creates a new client for the given hub.
func (h *hub) NewClient() *Client {
    return Client(h)
}

// findAndMergeTopic finds existing topics and merges new ones into the trie.
func (h *hub) findAndMergeTopic(topic string[]) (*topicNode, error) {
}
