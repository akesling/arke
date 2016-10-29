// interchange defines the core functionality of an Arke hub.
//
// It provides an API for creating a new hub, publishing and subscribing, as
// well as appropriate interfaces for implementing transparent hub Client
// wrapper types.
package interchange

import (
	"log"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"strings"
	"time"
)

const (
	// Number of dead subscribers per topic before we walk the subscriber list
	// and collect the garbage.
	collectionThreshold = 4

	subscriberBufferSize = 100
	hubBufferSize        = 100
)

// A hub plays matchmaker between publishers and subscribers.
//
// It contains the root of the topic Trie
//
// hub is thread-hostile, only one go-routine should access it at a time.
type hub struct {
	root *topicNode

	pub     chan *publication
	sub     chan *subscription
	cleanup chan []string
}

// NewHub builds an Arke hub.
func NewHub(ctx context.Context, cancel context.CancelFunc) *hub {
	log.Println("Creating new Arke hub.")
	new_root := newTopicNode(ctx, cancel, []string{rootName})

	h := &hub{
		root:    new_root,
		pub:     make(chan *publication, hubBufferSize),
		sub:     make(chan *subscription, hubBufferSize),
		cleanup: make(chan []string, hubBufferSize),
	}

	h.start()
	return h
}

// findTopic takes a topic name and returns the topic or an error if not found.
func (h *hub) findTopic(topic []string) (found *topicNode, err error) {
	found, rest, _ := h.root.MaybeFindTopic(topic)

	if len(rest) != 0 {
		return nil, errors.New(
			fmt.Sprintf("Topic not found: %s",
				strings.Join(topic, topicDelimeter)))
	}

	return found, nil
}

// findOrCreateTopic takes a topic name and returns or creates the topic.
//
// If creation fails, an error will be returned.
func (h *hub) findOrCreateTopic(topic []string) (found *topicNode, err error) {
	found, rest, _ := h.root.MaybeFindTopic(topic)

	if len(rest) != 0 {
		found, err = found.CreateChild(rest)
	}

	return found, err
}

func (h *hub) start() {
	log.Println("Starting Arke hub.")
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

				topic.Apply(nil, func(_, t *topicNode) {
					for i := range t.Subscribers {
						t.Subscribers[i].Send(&newPub.Message)
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
			case <-h.root.ctx.Done():
				cancelPendingSubscriptionRequests(h.sub);
				break event_loop
			}
		}
	}()
}

func cancelPendingSubscriptionRequests(subs chan*subscription) {
	for {
		select {
		case s := <-subs:
			close(s.Client)
		default:
			break
		}
	}
}

// Publish synchronously issues a publication request on the given topic.
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
		Topic:   strings.Split(topic, topicDelimeter),
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
func (h *hub) Subscribe(name, topic string, lease time.Duration) (<-chan Message, error) {
	select {
	case <-h.root.ctx.Done():
		return nil, errors.New("Cannot subscribe to a closed hub.");
	default:
	}

	deadline := time.Now().Add(lease)
	comm := make(chan Message)
	var expandedTopic []string

	if topic == rootName {
		expandedTopic = []string{rootName}
	} else {
		expandedTopic = strings.Split(topic, topicDelimeter)
	}

	h.sub <- &subscription{
		Topic:    expandedTopic,
		Name:     name,
		Deadline: deadline,
		Client:   comm,
	}

	return comm, nil
}

// NewClient creates a new Client for the given hub.
func NewClient(h *hub) Client {
	return Client(h)
}
