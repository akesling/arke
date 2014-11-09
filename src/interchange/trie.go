package interchange

import (
    "code.google.com/p/go.net/context"
    "errors"
    "fmt"
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

func newTopicNode(ctx context.Context, cancel context.CancelFunc, name []string) *topicNode {
    return &topicNode{
        ctx:            ctx,
        Cancel:         cancel,
        Name:           name,
        Children:       make([]*topicNode, 0, 10),
        Subscribers:    make([]*subscriber, 0, 10),
    }
}

func IsValidTopic(name []string) bool {
    for i := range name {
        token := name[i]
        switch (token) {
            case "", ".":
                return false
        }
    }

    return true
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
// be the parent of the topic if it did exist.  This returned node may require
// realigning its name, e.g. `rest` may be foo.bar.baz where the name of the
// returned node is foo.bar.qux.

// In the failure case, rest will be the remainder of the name string that
// wasn't found.
//
// Assumes `topic` is in canonical form (e.g. no empty elements or those of the
// form of topicDelimeter except in the case of a root topic).
// If a non-canonical topic is passed, no matching topic will be found.
// XXX(akesling): Properly allow caller to understand whether we found a
// "clean" parent or an overlapping "parent".
func (t *topicNode) MaybeFindTopic(topic []string) (nearestTopic *topicNode, rest []string, overlaps bool) {
    if len(topic) < 1 || t == nil {
        return t, []string{}, false
    }

    // TODO(akesling): Reimplement this as something better than a
    // linear search.
    nearestTopic = t
    head := topic[0]
    rest = topic
    for i := range t.Children {
        child := t.Children[i]

        if len(child.Name) < 1 {
            // Empty names can wreak terrible havoc on all
            // trie manipulations... thus this violates a
            // core invariant and we should die accordingly.
            panic(errors.New("Hub child has an empty name."))
        }

        if head == child.Name[0] {
            nearestTopic = child
            rest = topic[1:]

            for i := 1; i < len(child.Name) && i < len(topic); i += 1 {
                // Names partially overlap
                if topic[i] != child.Name[i] {
                    return nearestTopic, newCopy(topic), true
                }
                rest = topic[i+1:]
            }

            // The current child's name has been consumed and we should look at
            // its children to find more of our topic.
            if len(rest) != 0 {
                return nearestTopic.MaybeFindTopic(rest)
            }

            break
        }
    }

    return nearestTopic, newCopy(rest), false
}

func newCopy(name []string) []string {
    cp := make([]string, len(name))
    copy(cp, name)
    return cp
}

// If this child already exists, it's considered a no-op and CreateChild
// returns successfully with newTopic being the existing child.
//
// CreateChild has for resulting cases:
// 1) Error on invalid subTopic
// 2) Return existing topicNode
// 3) Create new topicNode in the trie
// 4) Modify trie structure in the process of 2 or 3
//
// Case 4 happens when we have an existing node with an overlapping run from our
// desired new topic name.  E.g. we have a foo.bar.baz node in the trie but we
// want to create a node foo.bar.qux.  In this case, we must create a foo.bar
// node with foo.bar.baz being renamed baz and assigned as a child of foo.bar.
// This requires rebinding a new context for baz and its subscribers and
// children that derives from the new foo.bar.  We then add a qux node that derives
// from foo.bar.
func (t *topicNode) CreateChild(subTopic []string) (newTopic *topicNode, err error) {
    if !IsValidTopic(subTopic) {
        return nil, errors.New(fmt.Sprintf(
            "Malformed topicName (%s) provided to CreateChild of topicNode (%s)",
            subTopic, t.Name))
    }

    // Empty sub-topic name -> return self
    if len(subTopic) == 0 {
        return t, nil
    }

    // Sub-topic already exists -> return sub-topic
    candidate, rest, overlaps := t.MaybeFindTopic(subTopic)
    if len(rest) == 0 {
        return candidate, nil
    }

    // MaybeFindTopic gives us the closest topicNode to our goal,
    // thus we may construct a child of that node with a name of
    // "rest" if the rest isn't overlapping.
    if !overlaps {
        child_ctx, cancel_child := context.WithCancel(candidate.ctx)
        new_topic_node := newTopicNode(child_ctx, cancel_child, rest)
        candidate.Children = append(candidate.Children, new_topic_node)

        return new_topic_node, nil
    }

    // Since they overlap, we need to do some trie surgery
    panic(errors.New("Overlapping not implemented yet."))
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
