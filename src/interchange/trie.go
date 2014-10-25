package interchange

import (
    "code.google.com/p/go.net/context"
    "log"
)

// A topicNode represents a vertex in the topic trie.
//
// Topic nodes occur when there exists a subscriber on a topic.  You can think
// of a topic as a word whose "characters" are delimited by periods. The topic
// trie as such acts as a trie over these tokens The root is implicitly named
// '.' and acts specially as it doesn't collapse, the following example will
// show this in practice.
//
// Example: for the trie consisting of subscribers on foo.bar.baz and
// foo.bar.qux there will be four topic nodes: one for the root (whose name is
// '.'), one for the branching node whose name is foo.bar and one node for each
// of baz and qux.
type topicNode struct {
    ctx         context.Context
    deadSubs    int

    Cancel      context.CancelFunc
    Name        []string
    Children    []*topicNode
    Subscribers []*subscriber
}

func newTopicNode(name []string) *topicNode {
    return &topicNode{
        Name:           name,
        Children:       make([]*topicNode, 0, 10),
        Subscribers:    make([]*subscriber, 0, 10),
    }
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

// MaybeFindTopic searches the given topic trie for the provided topic.
//
// An empty `rest` indicates success.
//
// If the topic isn't found, it returns the closest ancestor node to what would
// be the parent of the topic if it did exist.  In the failure case, rest will
// be the remainder of the name string that wasn't found.
//
// Assumes `topic` is in canonical form (e.g. no empty elements or those of the
// form of topicDelimeter except in the case of a root topic).
// If a non-canonical topic is passed, no matching topic will be found.
func (t *topicNode) MaybeFindTopic(topic []string) (nearestTopic *topicNode, rest []string) {
    if len(topic) < 1 || t == nil {
        return t, []string{}
    }

    // TODO(akesling): Reimplement this as something better than a
    // linear search.
    nearestTopic = t
    head := topic[0]
    rest = topic[1:]
    for i := range t.Children {
        child := t.Children[i]

        if len(child.Name) < 1 {
            log.Fatal("Hub child has an empty name.")
        }

        if head == child.Name[0] {
            nearestTopic = child

            for i := 1; i < len(child.Name) && i < len(topic); i += 1 {
                if topic[i] != child.Name[i] {
                    return child.MaybeFindTopic(topic[i:])
                }
                rest = topic[i+1:]
            }

            break
        }
    }

    return nearestTopic, rest
}

// If this child already exists, it's considered a no-op and CreateChild
// returns successfully with newTopic being the existing child.
func (t *topicNode) CreateChild(subTopic []string) (newTopic *topicNode, err error) {
    if len(subTopic) == 0 {
        return t, nil
    }

    //candidate, rest := t.MaybeFindTopic(subTopic)
    // TODO(akesling)
    log.Fatal("Not implemented yet!")
    return nil, nil
}

func (t *topicNode) Collapse() {
    t.CollapseSubscribers()

    // Recursively collapse children and clean up empty trie branches.
    for i := 0; i < len(t.Children); i += 1 {
        child := t.Children[0]
        child.Collapse()

        if len(child.Subscribers) == 0 && len(child.Children) == 0 {
            if len(t.Children) == 0 {
                break
            }

            var last uint
            last = uint(len(t.Children)-1)

            t.Children[i] = t.Children[last]
            // Remove the array's ref to this child so it can be collected.
            t.Children[last] = nil
            // Truncate slice to exclude now-defunct final element.
            t.Children = t.Children[0:last]

            // We've now moved an unseen element to this index... we should
            // properly evaluate it.
            i -= 1
        }
    }

    // Collapse unnecessary runs of subscriber-free nodes.
    if len(t.Subscribers) == 0 && len(t.Children) == 1 {
        child := t.Children[0]
        t.Children[0] = nil
        t.Children = t.Children[0:0]

        t.Name = append(t.Name, child.Name...)
        t.Subscribers = child.Subscribers
        t.Children = child.Children
    }
}

func (t *topicNode) CollapseSubscribers() {
    // Note: List length may change during iteration.
    for i := 0; i < len(t.Subscribers); i += 1 {
        select {
        case <-t.Subscribers[i].Done:
            if len(t.Subscribers) == 0 {
                break
            }

            var last uint
            last = uint(len(t.Subscribers)-1)

            t.Subscribers[i] = t.Subscribers[last]
            // Remove the array's ref to this subscriber so it can be collected.
            t.Subscribers[last] = nil
            // Truncate slice to exclude now-defunct final element.
            t.Subscribers = t.Subscribers[0:last]

            // We've now moved an unseen element to this index... we should
            // properly evaluate it.
            i -= 1
        default:
        }
    }
}

// Map recursively post-applies a function to all topicNodes in a
// topic trie rooted at the callee.
func (t *topicNode) Map(f func(*topicNode)) {
    for i := range t.Children {
        t.Children[i].Map(f)
    }
    f(t)
}
