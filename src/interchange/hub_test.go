package interchange

import (
	"code.google.com/p/go.net/context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
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
		_, err := h.findOrCreateTopic(grid[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	// New retrieve nodes using the already tested findOrCreateTopic
	// We can't save the node addresses above as they
	// may shuffle around during construction.
	for i := range grid {
		current_topic, err := h.findOrCreateTopic(grid[i])
		if err != nil {
			t.Fatal(err)
		}
		topics[i] = current_topic
	}

	// Test retrieval
	for i := range grid {
		topic_name := grid[i]
		expected_topic := topics[i]
		actual_topic, err := h.findTopic(topic_name)

		if err != nil {
			t.Error(err)
		}

		if expected_topic != actual_topic || !reflect.DeepEqual(expected_topic.Name, actual_topic.Name) {
			t.Error(
				fmt.Sprintf("For topic name %q, expected topic (%s) did not match actual topic found (%s).",
					topic_name,
					printTopicPointer(expected_topic),
					printTopicPointer(actual_topic)))
		}
	}
}

func benchmarkPublicationNTopicsMSubs(n, m int, b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	h := NewHub()
	h.Start(ctx)
	for i := 0; i < n; i++ {
		topic := fmt.Sprintf("foo.%d", i)
		for j := 0; j < m; j++ {
			go func() {
				sub, _ := h.Subscribe(fmt.Sprintf("%d", i), topic, time.Minute)
				for m := range sub {
					_ = m
				}
			}()
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Publish("foo", Message{
			Body: []byte("foo"),
		})
	}

	b.StopTimer()
	cancel()
}

func benchmarkPublicationNSubs(n int, b *testing.B) {
	benchmarkPublicationNTopicsMSubs(1, n, b)
}

func BenchmarkPublicationNoSubs(b *testing.B)     { benchmarkPublicationNSubs(0, b) }
func BenchmarkPublicationOneSub(b *testing.B)     { benchmarkPublicationNSubs(1, b) }
func BenchmarkPublication10Subs(b *testing.B)     { benchmarkPublicationNSubs(10, b) }
func BenchmarkPublication100Subs(b *testing.B)    { benchmarkPublicationNSubs(100, b) }
func BenchmarkPublication1000Subs(b *testing.B)   { benchmarkPublicationNSubs(1000, b) }
func BenchmarkPublication10000Subs(b *testing.B)  { benchmarkPublicationNSubs(10000, b) }
func BenchmarkPublication100000Subs(b *testing.B) { benchmarkPublicationNSubs(100000, b) }

func benchmarkPublicationNTopics(n int, b *testing.B) {
	benchmarkPublicationNTopicsMSubs(n, 1, b)
}

func BenchmarkPublicationNoTopics(b *testing.B)     { benchmarkPublicationNSubs(0, b) }
func BenchmarkPublicationOneTopic(b *testing.B)     { benchmarkPublicationNSubs(1, b) }
func BenchmarkPublication10Topics(b *testing.B)     { benchmarkPublicationNSubs(10, b) }
func BenchmarkPublication100Topics(b *testing.B)    { benchmarkPublicationNSubs(100, b) }
func BenchmarkPublication1000Topics(b *testing.B)   { benchmarkPublicationNSubs(1000, b) }
func BenchmarkPublication10000Topics(b *testing.B)  { benchmarkPublicationNSubs(10000, b) }
func BenchmarkPublication100000Topics(b *testing.B) { benchmarkPublicationNSubs(100000, b) }

func benchmarkPublicationToRandomOfNTopics(n int, b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	h := NewHub()
	h.Start(ctx)
	for i := 0; i < n; i++ {
		topic := fmt.Sprintf("foo.%d", i)
		for j := 0; j < 1; j++ {
			go func() {
				sub, _ := h.Subscribe(fmt.Sprintf("%d", i), topic, time.Minute)
				for m := range sub {
					_ = m
				}
			}()
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		topic := fmt.Sprintf("foo.%d", rand.Intn(n))
		h.Publish(topic, Message{
			Body: []byte("foo"),
		})
	}

	b.StopTimer()
	cancel()
}

func BenchmarkPublicationRandom10Topics(b *testing.B)  { benchmarkPublicationToRandomOfNTopics(10, b) }
func BenchmarkPublicationRandom100Topics(b *testing.B) { benchmarkPublicationToRandomOfNTopics(100, b) }
func BenchmarkPublicationRandom1000Topics(b *testing.B) {
	benchmarkPublicationToRandomOfNTopics(1000, b)
}
func BenchmarkPublicationRandom10000Topics(b *testing.B) {
	benchmarkPublicationToRandomOfNTopics(10000, b)
}
func BenchmarkPublicationRandom100000Topics(b *testing.B) {
	benchmarkPublicationToRandomOfNTopics(100000, b)
}
