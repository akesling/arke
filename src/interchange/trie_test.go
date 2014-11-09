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
            t.Error(fmt.Sprintf("For provided path (%q), MaybeFindTopic returned a name remainder (%q) when it should have been (%q).", path, rest_returned, rest_expected))
        }
    }

    // Find exact topics
    /******************/
    path := []string{"foo"}
    should_be_foo, rest, overlap := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo, _foo)
    rest_expectation(path, []string{}, rest)
    rest_expectation(path, []string{}, overlap)

    path = []string{"foo", "bar"}
    should_be_foo_bar, rest, overlap := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_bar, _foo_bar)
    rest_expectation(path, []string{}, rest)
    rest_expectation(path, []string{}, overlap)

    path = []string{"foo", "baz"}
    should_be_foo_baz, rest, overlap := root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_baz, _foo_baz)
    rest_expectation(path, []string{}, rest)
    rest_expectation(path, []string{}, overlap)

    path = []string{"qux"}
    should_be_qux, rest, overlap := root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux, _qux)
    rest_expectation(path, []string{}, rest)
    rest_expectation(path, []string{}, overlap)

    path = []string{"qux", "foo", "bar"}
    should_be_qux__foo_bar, rest, overlap := root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
    rest_expectation(path, []string{}, rest)
    rest_expectation(path, []string{}, overlap)

    // Return rest with found parent
    /******************************/
    path = []string{"foo", "bar", "baz"}
    should_be_foo_bar, rest, overlap = root.MaybeFindTopic(path)
    child_expectation(path, should_be_foo_bar, _foo_bar)
    rest_expectation(path, []string{"baz"}, rest)
    rest_expectation(path, []string{}, overlap)

    // Return rest with overlapping parent
    /************************************/
    path = []string{"qux", "foo", "baz"}
    should_be_qux__foo_bar, rest, overlap = root.MaybeFindTopic(path)
    child_expectation(path, should_be_qux__foo_bar, _qux__foo_bar)
    rest_expectation(path, []string{"foo", "baz"}, rest)
    rest_expectation(path, []string{"foo"}, overlap)
}

func TestCreateChild(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    tNode := newTopicNode(ctx, cancel, []string{"foo"})

    bar_baz_path := []string{"bar", "baz"}
    bar_baz, err := tNode.CreateChild(bar_baz_path)
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }
    if !reflect.DeepEqual(bar_baz.Name, []string{"bar", "baz"}) {
        t.Error(fmt.Sprintf("CreateChild returned child with incorrect name (%q) vs. expected (%q).", bar_baz.Name, bar_baz_path))
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(tNode.Children)))
    }
    if child_node := tNode.Children[0]; bar_baz != child_node {
        t.Error(fmt.Sprintf("CreateChild returned a node (%+v) other than the one which it stored (%+v).", bar_baz, child_node))
    }

    should_be_bar_baz, err := tNode.CreateChild(bar_baz_path)
    if err != nil {
        t.Error(fmt.Sprintf("CreateChild returned error when no-op was expected: %s", err))
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %d children when 1 was expected.", len(tNode.Children)))
    }
    if should_be_bar_baz != bar_baz {
        t.Error(fmt.Sprintf("CreateChild returned a new node (%+v) when returning an existing node was expected (%+v)", should_be_bar_baz, bar_baz))
    }
    if should_be_bar_baz != tNode.Children[0] {
        t.Error("CreateChild returned a node other than the one which it stored.")
    }

    bar_qux_path := []string{"bar", "qux"}
    bar_qux, err := tNode.CreateChild(bar_qux_path)
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }
    if !reflect.DeepEqual(bar_qux.Name, bar_qux_path) {
        t.Error(fmt.Sprintf("CreateChild returned child with incorrect name (%q) vs. expected (%q).", bar_baz.Name, bar_qux_path))
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created a node unexpectedly at a higher level. The relative root had 1 child, and now has %s.", len(tNode.Children)))
    }
    if bar_baz == bar_qux {
        t.Error(fmt.Sprintf("CreateChild returned an existing node (%+v) when returning a new node was expected (%+v)", bar_baz, bar_qux))
    }
    if !reflect.DeepEqual(tNode.Children[0].Name, []string{"bar"}) {
        t.Error(fmt.Sprintf("CreateChild did not name the trie branch correctly.  Expected [\"bar\"] and got %q", tNode.Children[0].Name))
        return
    }
    if bar_baz == tNode.Children[0].Children[0] && bar_qux == tNode.Children[0].Children[1] {
        children := make([]topicNode, len(tNode.Children[0].Children))
        for i := range tNode.Children[0].Children {
            children[i] = *tNode.Children[0].Children[i]
        }
        t.Error(fmt.Sprintf("CreateChild incorrectly expanded the trie structure. Children of new branch are %+v", children))
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
