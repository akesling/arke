package interchange

import (
	"fmt"
	"testing"
)

func TestFindOrCreateTopic(t *testing.T) {
	grid := [][]string{
		[]string{"foo"},
		[]string{"foo", "bar"},
		[]string{"foo", "qux"},
		[]string{"baz"},
	}

	h := NewHub()

	for i := range grid {
		topic_name := grid[i]
		node, err := h.findOrCreateTopic(topic_name)
		if err != nil && node != nil {
			t.Error("findOrCreateTopic returned error when topicNode creation was expected.")
		}

		node_again, err := h.findOrCreateTopic(topic_name)
		if err != nil && node_again != nil {
			t.Error(fmt.Sprintf("For name %q, findOrCreateTopic returned error when topicNode should have been found.", topic_name))
		}
		if node != node_again {
			t.Error(fmt.Sprintf("For name %q, findOrCreateTopic did not return the same topicNode on first retrieval after creation.", topic_name))
		}

		node_again, err = h.findOrCreateTopic(topic_name)
		if node != node_again {
			t.Error(fmt.Sprintf("For name %q, findOrCreateTopic did not return the same topicNode on repeated retrieval.", topic_name))
		}
	}
}

func printTopicPointer(topic *topicNode) string {
	return fmt.Sprintf("%p :: %+v", topic)
}

func TestFindTopic(t *testing.T) {
	// Topics to create and then find.
	grid := [][]string{
		[]string{"foo", "bar", "baz"},
		[]string{"foo", "bar", "qux"},
		[]string{"foo", "quuz"},
	}

	// Build trie
	h := NewHub()
	topics := make([]*topicNode, len(grid))
	for i := range grid {
		current_topic, err := h.findOrCreateTopic(grid[i])
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("Creating topic %q\n", grid[i])
		fmt.Print(h.root.RenderTrie())
		topics[i] = current_topic
	}

	fmt.Println("Rendering trie")
	fmt.Print(h.root.RenderTrie())

	// Test retrieval
	for i := range grid {
		topic_name := grid[i]
		expected_topic := topics[i]
		actual_topic, err := h.findTopic(topic_name)

		if err != nil {
			t.Error(err)
		}

		if expected_topic != actual_topic {
			t.Error(
				fmt.Sprintf("For topic name %q, expected topic (%s) did not match actual topic found (%s).",
					topic_name,
					printTopicPointer(expected_topic),
					printTopicPointer(actual_topic)))
		}
	}
}
