package interchange

import (
    "code.google.com/p/go.net/context"
    "fmt"
    "reflect"
    "testing"
)

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

    rest_expectation := func(path, rest_expected, rest_returned []string) {
        if !reflect.DeepEqual(rest_expected, rest_returned) {
            t.Error(fmt.Sprintf("For provided path (%v), MaybeFindTopic returned a name remainder (%v) when it should have been (%v).", path, rest_expected, rest_returned))
        }
    }

    // Find exact topics
    path := []string{"foo"}
    should_be_foo, rest := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo, _foo)
    rest_expectation(path, []string{}, rest)

    path = []string{"foo", "bar"}
    should_be_foo_bar, rest := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_bar, _foo_bar)
    rest_expectation(path, []string{}, rest)

    path = []string{"foo", "baz"}
    should_be_foo_baz, rest := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_baz, _foo_baz)
    rest_expectation(path, []string{}, rest)

    path = []string{"qux"}
    should_be_qux, rest := root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux, _qux)
    rest_expectation(path, []string{}, rest)

    path = []string{"qux", "foo", "bar"}
    should_be_qux__foo_bar, rest := root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
    rest_expectation(path, []string{}, rest)

    // Return rest with found parent
    path = []string{"foo", "bar", "baz"}
    should_be_foo_bar, rest = root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_bar, _foo_bar)
    rest_expectation(path, []string{"bar"}, rest)

    // Return rest with overlapping parent
    t.Error("Overlapping return value provides an inconsistent state.  This
    should be fixed so that caller understands when something is
    overlapping....")
    path = []string{"qux", "foo", "baz"}
    should_be_qux__foo_bar, rest = root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
    rest_expectation(path, []string{"baz"}, rest)
}

func TestCreateChild(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    tNode := newTopicNode(ctx, cancel, []string{"foo"})

    bar_baz, err := tNode.CreateChild([]string{"bar", "baz"})
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(tNode.Children)))
    }
    if child_node := tNode.Children[0]; bar_baz != child_node {
        t.Error(fmt.Sprintf("CreateChild returned a node (%+v) other than the one which it stored (%+v).", bar_baz, child_node))
    }

    bar_baz2, err := tNode.CreateChild([]string{"bar", "baz"})
    if err != nil {
        t.Error("CreateChild returned error when no-op was expected.")
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(tNode.Children)))
    }
    if bar_baz2 != bar_baz {
        t.Error("CreateChild returned a new node when returning an existing node was expected")
    }
    if bar_baz2 != tNode.Children[0] {
        t.Error("CreateChild returned a node other than the one which it stored.")
    }

    bar_qux, err := tNode.CreateChild([]string{"bar", "qux"})
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild modified the direct children of the node unexpectedly.  It has 1 child, and now has %s.", len(tNode.Children)))
    }
    if bar_baz != bar_qux {
        t.Error("CreateChild returned an existing node when returning a new node was expected")
    }
    if bar_baz == tNode.Children[0] || bar_qux == tNode.Children[0] {
        t.Error("CreateChild did not properly create a new trie branching node.")
    }
    if !reflect.DeepEqual(tNode.Children[0].Name, []string{"bar"}) {
        t.Error(fmt.Sprintf("CreateChild did not name the trie branch correctly.  Expected [bar] and got %s", tNode.Children[0].Name))
    }
    if bar_baz != tNode.Children[0].Children[0] {
        t.Error("CreateChild incorrectly expanded the trie structure.")
    }
    if bar_qux != tNode.Children[0].Children[1] {
        t.Error("CreateChild incorrectly expanded the trie structure.")
    }
}

func TestAddSub(t *testing.T) {
    /*
    topic := []string{"foo"}
    root := new(topicNode)
    newNode, err := root.CreateChild(topic)
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }

    if len(root.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %s children when 1 was expected.", len(root.Children)))
    }
    if newNode != root.Children[0] {
        t.Error("CreateChild returned a node other than the one which it stored.")
    }
    */
}
