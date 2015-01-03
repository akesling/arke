package interchange

import (
	"code.google.com/p/go.net/context"
	"errors"
	"fmt"
	"strings"
)

const (
	// Delimiter between topic elements.
	topicDelimeter = "."

	// Name of the Arke hub's root topic.
	rootName = "."
)

func copyTopic(topic []string) []string {
	cp := make([]string, len(topic))
	copy(cp, topic)
	return cp
}

func IsValidTopic(topic []string) bool {
	if len(topic) == 0 {
		return false
	}

	for i := range topic {
		token := topic[i]
		switch token {
		case "", rootName:
			return false
		}
	}

	return true
}

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
	ctx      context.Context
	deadSubs int

	Cancel      context.CancelFunc
	Name        []string
	Children    []*topicNode
	Subscribers []*subscriber
}

func newTopicNode(ctx context.Context, cancel context.CancelFunc, name []string) *topicNode {
	return &topicNode{
		ctx:         ctx,
		Cancel:      cancel,
		Name:        copyTopic(name),
		Children:    make([]*topicNode, 0, 10),
		Subscribers: make([]*subscriber, 0, 10),
	}
}

// AddSub creates a subscriber, attaches it to this topicNode and starts it.
//
// Parameters:
//  sub: The subscription from which to build the new Subscriber
//  cleanup: The channel via which the subscriber's 'death' will notify the hub.
func (t *topicNode) AddSub(sub *subscription, cleanup chan<- []string) *subscriber {
	ctx, cancel := context.WithDeadline(t.ctx, sub.Deadline)
	comm := make(chan Message)

	go func(ctx context.Context, source <-chan Message) {
	event_loop:
		for {
			select {
			case message := <-source:
				sub.Client <- message
			case <-ctx.Done():
				close(sub.Client)
				break event_loop
			}
		}
	}(ctx, comm)

	// When the subscriber is done, notify the hub for garbage collection.
	go func(topic []string, notify chan<- []string) {
		<-ctx.Done()
		notify <- topic
	}(sub.Topic, cleanup)

	new_subscriber := &subscriber{
		ctx:    ctx,
		Cancel: cancel,
		Done:   ctx.Done(),
		Name:   sub.Name,
		Sink:   comm,
	}
	t.Subscribers = append(t.Subscribers, new_subscriber)

	return new_subscriber
}

// MaybeFindTopic searches the given topic trie for the provided topic.
//
// An empty `rest` indicates success.
//
// If the topic isn't found, it returns the closest ancestor node to what would
// be the parent of the topic if it did exist.  This returned node may require
// realigning its name, e.g. `rest` may be foo.bar.baz where the name of the
// returned node is foo.bar.qux.
//
// In the failure case, rest will be the remainder of the name string that
// wasn't found.
//
// Assumes `topic` is in canonical form (e.g. no empty elements or those of the
// form of topicDelimeter except in the case of a root topic).
// If a non-canonical topic is passed, no matching topic will be found.
func (t *topicNode) MaybeFindTopic(topic []string) (nearestTopic *topicNode, rest, overlap []string) {
	if len(topic) < 1 || t == nil {
		return t, []string{}, []string{}
	}

	// TODO(akesling): Reimplement this as something better than a
	// linear search.
	nearestTopic = t
	rest = topic
	overlap = []string{}

	if t.Name[0] != rootName {
		i := 0
		for i < len(t.Name) && i < len(topic) {
			if topic[i] != t.Name[i] {
				break
			}
			i += 1
		}
		overlap = topic[:i]
	} else if len(t.Name) > 1 {
		// The root node must have a name of at most length 1.
		panic(
			errors.New(
				fmt.Sprintf(
					"Hub root has name longer than 1 component: %q", t.Name)))
	}

	if len(overlap) == len(topic) {
		return nearestTopic, []string{}, []string{}
	}

	for i := range t.Children {
		child := t.Children[i]

		if len(child.Name) < 1 {
			// Empty names can wreak terrible havoc on all
			// trie manipulations... thus this violates a
			// core invariant and we should die accordingly.
			panic(
				errors.New(
					fmt.Sprintf(
						"Hub child has an empty name.\n Printout of Trie:\n%s", t.RenderTrie())))
		}

		if topic[len(overlap)] == child.Name[0] {
			nearestTopic = child
			rest = rest[len(overlap):]
			overlap = []string{child.Name[0]}

			for i := 1; i < len(nearestTopic.Name) && i < len(rest); i += 1 {
				// Names partially overlap
				if rest[i] != nearestTopic.Name[i] {
					return nearestTopic, copyTopic(rest), copyTopic(rest[:i])
				}
			}

			// The current child's name has been consumed and we should look at
			// its children to find more of our topic.
			if len(rest)-len(overlap) != 0 {
				return nearestTopic.MaybeFindTopic(rest)
			} else {
				return nearestTopic, []string{}, []string{}
			}
		}
	}

	return nearestTopic, copyTopic(rest), copyTopic(overlap)
}

// If this child already exists, it's considered a no-op and CreateChild
// returns successfully with newTopic being the existing child.
//
// CreateChild has four resulting cases:
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
	candidate, rest, overlap := t.MaybeFindTopic(subTopic)
	if len(rest) == 0 {
		return candidate, nil
	}

	parent := candidate
	if len(overlap) > 0 && len(overlap) < len(candidate.Name) {
		// Partial overlap exists, so we must split the found candidate.
		ctx, cancel := context.WithCancel(parent.ctx)
		new_child := newTopicNode(ctx, cancel, parent.Name[len(overlap):])
		parent.Name = copyTopic(overlap)

		new_child.Children = parent.Children
		parent.Children = make([]*topicNode, 0, 10)

		new_child.Subscribers = parent.Subscribers
		parent.Subscribers = make([]*subscriber, 0, 10)

		reseatTopicNode(parent, new_child)
	}

	child_ctx, cancel_child := context.WithCancel(parent.ctx)
	new_topic_node := newTopicNode(child_ctx, cancel_child, rest[len(overlap):])
	parent.Children = append(parent.Children, new_topic_node)

	return new_topic_node, nil
}

// reseatTopicNode moves `to_be_reseated` from its current parent onto `parent`.
func reseatTopicNode(parent, to_be_reseated *topicNode) {
	parent.Children = append(parent.Children, to_be_reseated)
	to_be_reseated.Apply(parent, func(parent, child *topicNode) {
		child.ctx, child.Cancel = context.WithCancel(parent.ctx)
		for i := range child.Subscribers {
			sub := child.Subscribers[i]
			deadline, _ := sub.ctx.Deadline()
			sub.ctx, sub.Cancel = context.WithDeadline(child.ctx, deadline)
			sub.Done = sub.ctx.Done()
		}
	})
}

// Collapse is the effective garbage collection operation for a topic trie.
//
// It culls dead subscribers and topicNodes which no longer have any
// subscribers.  Recursively collapses children and cleans up empty trie
// branches.
func (t *topicNode) Collapse() {
	t.CollapseSubscribers()

	// If no children, skip.
	for i := 0; i < len(t.Children); i += 1 {
		child := t.Children[i]
		child.Collapse()

		if len(child.Subscribers) == 0 && len(child.Children) == 0 {
			last := uint(len(t.Children) - 1)

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
	// Leave the root node strictly named ["."].
	//
	// Note that reseating is unnecessary here as the context-hierarchy is
	// preserved in the elision of an intervening descendant; e.g. the removal
	// of B in A->B->C => A->C still preserves the property that cancellation of
	// A results in the cancellation of C.
	if t.Name[0] != rootName && len(t.Subscribers) == 0 && len(t.Children) == 1 {
		child := t.Children[0]
		t.Name = append(t.Name, child.Name...)
		t.Subscribers = child.Subscribers
		t.Children = child.Children
	}
}

func (t *topicNode) CollapseSubscribers() {
	// Note: List length may change during iteration.
	// If no subscribers, skip as we're completely collapsed.
	for i := 0; i < len(t.Subscribers); i += 1 {
		select {
		case <-t.Subscribers[i].Done:
			var last uint
			last = uint(len(t.Subscribers) - 1)

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

// Apply recursively pre-applies a function to all topicNodes in a
// topic trie rooted at the callee.
func (t *topicNode) Apply(parent *topicNode, f func(parent, child *topicNode)) {
	f(parent, t)
	for i := range t.Children {
		t.Children[i].Apply(t, f)
	}
}

func (t *topicNode) render(indentation string) string {
	var tokens []string

	tokens = append(tokens, indentation+strings.Join(t.Name, topicDelimeter)+topicDelimeter)

	// Partition children by "structural type"
	var leaves []*topicNode
	var sub_tries []*topicNode
	for i := range t.Children {
		child := t.Children[i]
		if len(child.Children) > 0 {
			sub_tries = append(sub_tries, child)
		} else {
			leaves = append(leaves, child)
		}
	}

	// Render leaves on one line
	if len(leaves) > 0 {
		var leaf_line []string
		for i := range leaves {
			leaf_line = append(leaf_line, strings.Join(leaves[i].Name, topicDelimeter))
		}
		tokens = append(tokens, indentation+"\t"+strings.Join(leaf_line, ", "))
	}

	// Render non-leaf children recursively
	for i := range sub_tries {
		tokens = append(tokens, sub_tries[i].render(indentation+"\t"))
	}

	return strings.Join(tokens, "\n")
}

func (t *topicNode) RenderTrie() string {
	return t.render("") + "\n"
}
