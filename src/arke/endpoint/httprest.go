package endpoint

import (
	"arke/interchange"
	"net/http"
	"time"
)

type httprest struct {
	hub   *interchange.Client
	Mux   *http.ServeMux
	codex Codex
}

func (h *httprest) Start(port int) (done <-chan struct{}) {
	// TODO(akesling): start the http server and hold a handle to stop it later.
}

func (h *httprest) Stop() {
	// TODO(akesling): actually stop the server... we can start
}

func (h *httprest) Publish(topic string, message Message) error {
	h.hub.Publish(topic, message)
}

func (h *httprest) Subscribe(name, topic string, lease time.Duration) (<-chan Message, error) {
	messages, err := h.hub.Subscribe(name, topic, lease)

	if err != nil {
		// Log this error condition and return the error wrapped appropriately.
	}

	go func() {
		for {
			// When we get a message, stick it in our buffer... evicting any
			//
		}
	}()
}

func NewEndpoint(hub, codex) *Endpoint {
	endpoint := &httprest{hub, http.NewServeMux(), http.codex}

	endpoint.Mux.HandleFunc("subscriptions", func(w http.ResponseWriter, r *http.Request) {
		// TODO(akesling): extract topic from URL, subscriber (which is the URL
		// to which we respond) and lease from the body.
		var topic string
		var subscriber string
		var lease time.Duration
		endpoint.Subscribe(subscriber, topic, lease)
	})

	endpoint.Mux.HandleFunc("topics", func(w http.ResponseWriter, r *http.Request) {
		// TODO(akesling): extract topic from URL, and message from body.
		var topic string
		var message interface{}
		endpoint.Publish(topic, message)
	})

	return Endpoint(endpoint)
}
