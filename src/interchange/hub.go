package interchange

struct Message {
}

struct publisher {
    name     string
    sink     <-chan Message
}

struct subscriber {
    name     string
    sink     ->chan Message
}

struct topicNode {
    name     string
    children topicNode*[]
    subs     subscriber[]
}

struct hub {
    topicTrie topicNode*
}

func NewHub() hub* {
    return new hub{}
}

func (h *hub) NewClient() {
}

func (h *hub) findAndMergeTopic(topic string[]) (subscriber[]*, err) {

}
