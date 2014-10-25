package interchange

import (
    "testing"
)

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
