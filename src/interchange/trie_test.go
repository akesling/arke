package interchange

import (
    "fmt"
    "reflect"
    "testing"
)

func TestCreateChild(t *testing.T) {
    tNode := newTopicNode([]string{"foo"})

    bar_baz, err := tNode.CreateChild([]string{"bar", "baz"})
    if err != nil {
        t.Error("CreateChild returned error when topicNode creation was expected.")
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %s children when 1 was expected.", len(tNode.Children)))
    }
    if bar_baz != tNode.Children[0] {
        t.Error("CreateChild returned a node other than the one which it stored.")
    }

    bar_baz2, err := tNode.CreateChild([]string{"bar", "baz"})
    if err != nil {
        t.Error("CreateChild returned error when no-op was expected.")
    }
    if len(tNode.Children) != 1 {
        t.Error(fmt.Sprintf("CreateChild created %s children when 1 was expected.", len(tNode.Children)))
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
        t.Error(fmt.Sprintf("CreateChild did not name the trie branch correctly.  Expected [\"bar\"] and got %s", tNode.Children[0].Name))
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
