package endpoint

import (
	"arke/interchange"
	"net/http"
	"time"
)

const (
	maxLeaseDuration = time.Duration(5) * time.Minute()
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

func (h *httprest) Subscribe(subscriberURL, topic string, lease time.Duration) (<-chan Message, error) {
	messages, err := h.hub.Subscribe(subscriberURL, topic, lease)

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

func constrainLease(requestedLease time.Duration) time.Duration {
	switch requestedLease {
	case requestedLease > maxLeaseDuration:
		return maxLeaseDuration
	case requestedLease < time.Duration:
		return time.Duration
	default:
		return requestedLease
	}
}

func decodeTopicURLPath(path string) (topic string) {
	tokens := strings.Split("/", path)
	for i := range tokens {
		tokens[i] = url.QueryUnescape(tokens[i])
	}
	topic = strings.Join(".", tokens)
}

func NewEndpoint(hub, codex) *Endpoint {
	endpoint := &httprest{hub, http.NewServeMux(), http.codex}

	endpoint.Mux.HandleFunc("subscriptions", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case POST:
			topic := decodeTopicURLPath(request.Opaque)

			requestFields, err := codex.Decode(request.Body)
			if err != nil || !requestFields.(map[string]string) {
				rw.WriteHeader(http.StatusMalformed)
				return
			}

			var subscriberURL string
			if subscriberURL, ok := requestFields["address"]; !ok {
				rw.WriteHeader(http.StatusMalformed)
				return
			}

			var requestedLease string
			if requestedLease, ok := requestFields["lease_duration"]; !ok {
				rw.WriteHeader(http.StatusMalformed)
				return
			}

			actualLease := constrainLease(time.Duration() * time.Second(strconv.ParseInt(requestedLease, 10, 64)))
			// Leases with the nil duration shouldn't _do_ anything.
			if actualLease == 0 {
				rw.WriteHeader(http.StatusCreated)
				rw.Write(codex.Encode(map[string]string{"lease_duration": "0"}))
				return
			}

			messages, err := endpoint.Subscribe(subscriber, topic, lease)
			if err != nil {
				rw.WriterHeader(http.StatusForbidden)
				rw.Write(codex.Encode(map[string]string{"error_message": err.Error()}))
				return
			}

			rw.WriterHeader(http.StatusCreated)
			rw.Write(codex.Encode(
				map[string]string{
					"lease_duration": fmt.Sprintf("%d", actualLease/time.Second()),
				}))
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
			// TODO(akesling): include the appropriate Allow header.
		}
	})

	endpoint.Mux.HandleFunc("topics", func(rw http.ResponseWriter, request *http.Request) {
		topic := decodeTopicURLPath(request.Opaque)
		message := codex.Decode(request.Body)

		switch request.Method {
		case POST:
			err := endpoint.Publish(topic, message)
			if err != nil {
				rw.WriterHeader(http.StatusForbidden)
				rw.Write(codex.Encode(map[string]string{"error_message": err.Error()}))
				return
			}

			rw.WriterHeader(http.StatusCreated)
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
			// TODO(akesling): include the appropriate Allow header.
		}
	})

	return Endpoint(endpoint)
}
