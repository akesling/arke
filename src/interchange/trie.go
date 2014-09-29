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

// If this child already exists, it's considered a no-op and CreateChild
// returns successfully with newTopic being the existing child.
func (t *topicNode) CreateChild(subTopic []string) (newTopic *topicNode, err error) {
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

        t.Name = append(t.Name, child.Name)
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
