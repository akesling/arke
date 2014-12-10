package interchange

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestIsValidTopic(t *testing.T) {
	expect_true := func(b bool, msg string) {
		if !b {
			t.Error(msg)
		}
	}

	expect_valid := func(topic []string) {
		expect_true(IsValidTopic(topic), fmt.Sprintf("Should have been valid: %q", topic))
	}
	expect_not_valid := func(topic []string) {
		expect_true(!IsValidTopic(topic), fmt.Sprintf("Should _not_ have been valid: %q", topic))
	}

	expect_valid([]string{"foo"})
	expect_valid([]string{"foo", "bar", "baz", "qux", "quz"})

	expect_not_valid([]string{})
	expect_not_valid([]string{""})
	expect_not_valid([]string{"."})
}

func TestAddSub(t *testing.T) {
	root_ctx, cancel_root := context.WithCancel(context.Background())
	root := newTopicNode(root_ctx, cancel_root, []string{"."})

	topic := []string{"foo", "bar"}
	new_node, _ := root.CreateChild(topic)

	death_notifications := make(chan []string)
	client_messages := make(chan Message)

	new_subscriber := new_node.AddSub(&subscription{
		Topic:    topic,
		Name:     "source",
		Deadline: time.Now().Add(time.Minute * 20),
		Client:   client_messages,
	}, death_notifications)

	for i := 0; i < 10; i += 1 {
		message_sent := Message{Source: fmt.Sprintf("test_%d", i)}
		new_subscriber.Sink <- message_sent

		select {
		case message_received := <-client_messages:
			if !reflect.DeepEqual(message_sent, message_received) {
				t.Error(fmt.Sprintf("(%d): Message received (%+v) did not match message expected (%+v)", i, message_received, message_sent))
			}
		default:
			t.Error(fmt.Sprintf("(%d): Expected message never received from subscriber", i))
		}
	}

	cancel_root()
	select {
	case notified_topic := <-death_notifications:
		if !reflect.DeepEqual(notified_topic, topic) {
			t.Error(fmt.Sprintf("Expected topic (%q) does not equal topic notified (%q)", topic, notified_topic))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for notification of subscriber death.")
		return
	}

	select {
	case <-client_messages:
	default:
		t.Error("Client channel not closed by dying subscriber")
	}
}

func TestMaybeFindTopic(t *testing.T) {
	// Build a sample topicTrie
	ctx, cancel := context.WithCancel(context.Background())
	root := newTopicNode(ctx, cancel, []string{"."})

	add_child := func(parent *topicNode, new_child *topicNode) {
		parent.Children = append(parent.Children, new_child)
	}

	_foo := newTopicNode(ctx, cancel, []string{"foo"})
	_foo_bar := newTopicNode(ctx, cancel, []string{"bar"})
	add_child(_foo, _foo_bar)
	_foo_baz := newTopicNode(ctx, cancel, []string{"baz"})
	add_child(_foo, _foo_baz)
	add_child(root, _foo)

	_qux := newTopicNode(ctx, cancel, []string{"qux"})
	_qux__foo_bar := newTopicNode(ctx, cancel, []string{"foo", "bar"})
	add_child(_qux, _qux__foo_bar)
	add_child(root, _qux)

	child_expectation := func(path []string, node_returned, node_expected *topicNode) {
		if node_expected != node_returned {
			t.Error(fmt.Sprintf("For provided path (%v), MaybeFindTopic returned a node (%+v) other than the one which it should have (%+v).", path, node_returned, node_expected))
		}
	}

	path_expectation := func(path, rest_expected, rest_returned []string) {
		if !reflect.DeepEqual(rest_expected, rest_returned) {
			t.Error(fmt.Sprintf("For provided path (%q), MaybeFindTopic returned a name remainder (%q) when it should have been (%q).", path, rest_returned, rest_expected))
		}
	}

	// Find exact topics
	/******************/
	path := []string{"foo"}
	should_be_foo, rest, overlap := root.MaybeFindTopic(path)
	child_expectation(path, should_be_foo, _foo)
	path_expectation(path, []string{}, rest)
	path_expectation(path, []string{}, overlap)

	path = []string{"foo", "bar"}
	should_be_foo_bar, rest, overlap := root.MaybeFindTopic(path)
	child_expectation(path, should_be_foo_bar, _foo_bar)
	path_expectation(path, []string{}, rest)
	path_expectation(path, []string{}, overlap)

	path = []string{"foo", "baz"}
	should_be_foo_baz, rest, overlap := root.MaybeFindTopic(path)
	child_expectation(path, should_be_foo_baz, _foo_baz)
	path_expectation(path, []string{}, rest)
	path_expectation(path, []string{}, overlap)

	path = []string{"qux"}
	should_be_qux, rest, overlap := root.MaybeFindTopic(path)
	child_expectation(path, should_be_qux, _qux)
	path_expectation(path, []string{}, rest)
	path_expectation(path, []string{}, overlap)

	path = []string{"qux", "foo", "bar"}
	should_be_qux__foo_bar, rest, overlap := root.MaybeFindTopic(path)
	child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
	path_expectation(path, []string{}, rest)
	path_expectation(path, []string{}, overlap)

	// Return rest with found parent
	/******************************/
	path = []string{"foo", "bar", "baz"}
	should_be_foo_bar, rest, overlap = root.MaybeFindTopic(path)
	child_expectation(path, should_be_foo_bar, _foo_bar)
	path_expectation(path, []string{"baz"}, rest)
	path_expectation(path, []string{}, overlap)

	// Return rest with overlapping parent
	/************************************/
	path = []string{"qux", "foo", "baz"}
	should_be_qux__foo_bar, rest, overlap = root.MaybeFindTopic(path)
	child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
	path_expectation(path, []string{"foo", "baz"}, rest)
	path_expectation(path, []string{"foo"}, overlap)
}

func TestCreateChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := newTopicNode(ctx, cancel, []string{"."})

	bar_baz_path := []string{"bar", "baz"}
	bar_baz, err := root.CreateChild(bar_baz_path)
	if err != nil {
		t.Error("CreateChild returned error when topicNode creation was expected.")
	}
	if !reflect.DeepEqual(bar_baz.Name, []string{"bar", "baz"}) {
		t.Error(fmt.Sprintf("CreateChild returned child with incorrect name (%q) vs. expected (%q).", bar_baz.Name, bar_baz_path))
	}
	if len(root.Children) != 1 {
		t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(root.Children)))
	}
	if child_node := root.Children[0]; bar_baz != child_node {
		t.Error(fmt.Sprintf("CreateChild returned a node (%+v) other than the one which it stored (%+v).", bar_baz, child_node))
	}

	should_be_bar_baz, err := root.CreateChild(bar_baz_path)
	if err != nil {
		t.Error(fmt.Sprintf("CreateChild returned error when no-op was expected: %s", err))
	}
	if len(root.Children) != 1 {
		t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(root.Children)))
	}
	if should_be_bar_baz != bar_baz {
		t.Error(fmt.Sprintf("CreateChild returned a new node (%+v) when returning an existing node was expected (%+v)", should_be_bar_baz, bar_baz))
	}
	if should_be_bar_baz != root.Children[0] {
		t.Error("CreateChild returned a node other than the one which it stored.")
	}

	bar_qux_path := []string{"bar", "qux"}
	bar_qux, err := root.CreateChild(bar_qux_path)
	if err != nil {
		t.Error("CreateChild returned error when topicNode creation was expected.")
	}
	if !reflect.DeepEqual(bar_qux.Name, bar_qux_path) {
		t.Error(fmt.Sprintf("CreateChild returned child with incorrect name (%q) vs. expected (%q).", bar_baz.Name, bar_qux_path))
	}
	if len(root.Children) != 1 {
		t.Error(fmt.Sprintf("CreateChild created a node unexpectedly at a higher level. The relative root had 1 child, and now has %d.", len(root.Children)))
	}
	if bar_baz == bar_qux {
		t.Error(fmt.Sprintf("CreateChild returned an existing node (%+v) when returning a new node was expected (%+v)", bar_baz, bar_qux))
	}
	if !reflect.DeepEqual(root.Children[0].Name, []string{"bar"}) {
		t.Error(fmt.Sprintf("CreateChild did not name the trie branch correctly.  Expected [\"bar\"] and got %q", root.Children[0].Name))
		return
	}
	if bar_baz == root.Children[0].Children[0] && bar_qux == root.Children[0].Children[1] {
		children := make([]topicNode, len(root.Children[0].Children))
		for i := range root.Children[0].Children {
			children[i] = *root.Children[0].Children[i]
		}
		t.Error(fmt.Sprintf("CreateChild incorrectly expanded the trie structure. Children of new branch are %+v", children))
	}
}

func TestReseatTopicNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := newTopicNode(ctx, cancel, []string{"."})

	foo, _ := root.CreateChild([]string{"foo"})
	foo_bar, _ := root.CreateChild([]string{"foo", "bar"})
	foo_bar_baz, _ := root.CreateChild([]string{"foo", "bar", "baz"})

	death_notifications := make(chan []string)

	client1_messages := make(chan Message)
	foo_bar_baz.AddSub(&subscription{
		Topic:    []string{"foo", "bar", "baz"},
		Name:     "flib",
		Deadline: time.Now().Add(time.Minute * 20),
		Client:   client1_messages,
	}, death_notifications)

	client2_messages := make(chan Message)
	foo_bar_baz.AddSub(&subscription{
		Topic:    []string{"foo", "bar", "baz"},
		Name:     "bilf",
		Deadline: time.Now().Add(time.Minute * 20),
		Client:   client2_messages,
	}, death_notifications)

	reseatTopicNode(foo, foo_bar_baz)
	if len(foo.Children) != 2 || foo.Children[0] != foo_bar || foo.Children[1] != foo_bar_baz {
		t.Error(
			fmt.Sprintf(
				"Expected foo to have the two children [%p %p], but it instead has %v",
				foo_bar, foo_bar_baz, foo.Children))
	}

	foo_bar.Cancel()
	select {
	case <-foo_bar_baz.ctx.Done():
		t.Error("foo_bar_baz's context was improperly reseated")
	case <-foo_bar_baz.Subscribers[0].Done:
		t.Error("foo_bar_baz's first subscriber's context was improperly reseated")
	case <-foo_bar_baz.Subscribers[1].Done:
		t.Error("foo_bar_baz's first subscriber's context was improperly reseated")
	default:
	}

	foo.Cancel()
	for _ = range []int{0, 1, 2} {
		select {
		case <-foo_bar_baz.ctx.Done():
		case <-foo_bar_baz.Subscribers[0].Done:
		case <-foo_bar_baz.Subscribers[1].Done:
		default:
			t.Error("foo_bar_baz or its subscribers were improperly reseated")
		}
	}
}

func TestCollapseSubscribers(t *testing.T) {
	// TODO(akesling)
}

func TestCollapse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := newTopicNode(ctx, cancel, []string{"."})

	root.CreateChild([]string{"foo", "bar", "baz", "qux"})
	root.CreateChild([]string{"foo", "bar", "baz", "quuz"})
	root.CreateChild([]string{"foo", "qux"})

	death_notifications := make(chan []string)

	client1_messages := make(chan Message)
	foo_baz_flibbity_blibbity_bop, _ := root.CreateChild([]string{"foo", "baz", "flibbity", "blibbity", "bop"})
	foo_baz_flibbity_blibbity_bop.AddSub(&subscription{
		Topic:    []string{"foo", "baz", "flibbity", "blibbity", "bop"},
		Name:     "source",
		Deadline: time.Now().Add(time.Minute * 20),
		Client:   client1_messages,
	}, death_notifications)

	client2_messages := make(chan Message)
	foo_quuz, _ := root.CreateChild([]string{"foo", "quuz"})
	foo_quuz.AddSub(&subscription{
		Topic:    []string{"foo", "quuz"},
		Name:     "source",
		Deadline: time.Now().Add(time.Minute * 20),
		Client:   client2_messages,
	}, death_notifications)

	root.Collapse()

	foo, _, _ := root.MaybeFindTopic([]string{"foo"})
	if !(len(foo.Name) == 1 && foo.Name[0] == "foo") {
		t.Error(fmt.Sprintf("Expected topic with name [\"foo\"], received topic with name %q", foo.Name))
	}

	type Expectation struct {
		Query []string
		Value *topicNode
	}

	topic_expectations := []Expectation{
		{[]string{"foo", "bar", "baz", "qux"}, foo},
		{[]string{"foo", "bar", "baz", "quuz"}, foo},
		{[]string{"foo", "qux"}, foo},
		{[]string{"foo", "qux"}, foo},
		{[]string{"foo", "quuz"}, foo_quuz},
		{[]string{"foo", "baz", "flibbity", "blibbity", "bop"}, foo_baz_flibbity_blibbity_bop},
	}

	for i := range topic_expectations {
		expectation := topic_expectations[i]
		found, _, _ := root.MaybeFindTopic(expectation.Query)
		if found != expectation.Value {
			t.Error(fmt.Sprintf("Topic found %+v was not the one expected %+v", found, expectation.Value))
		}
	}

	cancel()
	root.Collapse()
	if len(root.Children) > 0 {
		t.Error("Root has children, when the full trie should have collapsed")
	}
}
