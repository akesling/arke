package interchange

import (
    "fmt"
    "testing"
)

func testCreateChild(t *testing.T) {
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
}

func TestFindOrCreateTopic(t *testing.T) {
    topic := []string{"foo"}
    h := NewHub()

    firstNode, err := h.findOrCreateTopic(topic)
    if err != nil {
        t.Error("findOrCreateTopic returned error when topicNode creation was expected.")
    }

    firstNodeAgain, err := h.findOrCreateTopic(topic)
    if err != nil {
        t.Error("findOrCreateTopic returned error when topicNode should have been found.")
    }
    if firstNode != firstNodeAgain {
        t.Error("findOrCreateTopic did not return the same topicNode on first retrieval after creation.")
    }

    firstNodeAgain, err = h.findOrCreateTopic(topic)
    if firstNode != firstNodeAgain {
        t.Error("findOrCreateTopic did not return the same topicNode on repeated retrieval.")
    }
}
