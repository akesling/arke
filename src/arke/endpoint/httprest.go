package endpoint

import (
	"arke/interchange"
	"errors"
	"net/http"
	"time"
)

const (
	maxLeaseDuration = time.Duration(5) * time.Minute()
)

type httprest struct {
	hub    *interchange.Client
	port   int
	mux    *http.ServeMux
	server AsyncServer
	codex  *Codex
}

type AsyncServer http.Server

func (h *httprest) SetPort(port int) {
	h.port = port
}

func (h *httprest) Start() (done <-chan struct{}, err error) {
	// TODO(akesling): start the http server and hold a handle to stop it later.
	if port == 0 {
		return nil, errors.New("Port has not been set. Please call SetPort() with a valid port number.")
	}

	// TODO(akesling): if the server is already running, return an appropriate
	// error here.

	// TODO(akesling): implement AsyncServer to use http.Hijacker.Hijack to stop
	// the underlying http.Server goroutine.

	portString := fmt.Sprintf(":%d", h.port)
	if h.server == nil {
		h.server = &http.Server{
			Addr:           portString,
			Handler:        h.mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	} else {
		h.server.Addr = portString
	}

	err = h.server.ListenAndServe()
	if err != nil {
		return nil, err
	}
}

func (h *httprest) Stop() {
	h.server.Stop()
}

func (h *httprest) Publish(topic string, message Message) error {
	h.hub.Publish(topic, message)
}

func (h *httprest) Subscribe(subscriberURL, topic string, lease time.Duration) (<-chan Message, error) {
	// TODO(akesling): Add a GET call to the subscriberURL to verify validity
	// before adding a hub subscription.
	messages, err := h.hub.Subscribe(subscriberURL, topic, lease)

	if err != nil {
		// TODO(akesling): Log this error condition and return the error
		// wrapped appropriately.
	}

	go func() {
		client := &http.Client{}

	SubscribeLoop:
		for {

			// TODO(akesling):  Add message buffer eviction so we tolerate
			// misbehaving subscribers.
			//
			// When we get a message, stick it in our buffer... evicting any
			// messages that may have overstayed their welcome.
			select {
			case m := <-messages:
				encoded, err := h.codex.Transcode(m.Encoding, m.Body)
				if err != nil {
					// TODO(akesling): Log this error.
					continue
				}

				resp, err := client.Post(subscriberURL, h.codex.MIME(), encoded)
				defer resp.Body.Close()
				if err != nil {
					// TODO(akesling): Log this error.
					// Perhaps also account for high-error-rate subscribers.
					continue
				}
			case <-time.After(lease):
				encoded, err := h.codex.Encode(map[string]string{"status": "lease expired"})
				if err != nil {
					// TODO(akesling): Log this error.
				}

				resp, err := client.Put(subscriberURL, h.codex.MIME(), encoded)
				break SubscribeLoop
				// TODO(akesling): Handle server Stop() event.
				// case <-done:
				// break
			}
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

func NewEndpoint(hub, codex) *PortEndpoint {
	newMux := http.NewServeMux()
	endpoint := &httprest{hub, newMux, http.codex}

	newMux.HandleFunc("subscriptions", func(writer http.ResponseWriter, request *http.Request) {
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

			actualLease := constrainLease(
				time.Duration() * time.Second(strconv.ParseInt(requestedLease, 10, 64)))

			// Leases with the nil duration shouldn't _do_ anything.
			if actualLease == 0 {
				rw.WriteHeader(http.StatusCreated)
				rw.Write(codex.Encode(map[string]string{"lease_duration": "0"}))
				return
			}

			messages, err := endpoint.Subscribe(subscriberURL, topic, actualLease)
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

	newMux.HandleFunc("topics", func(rw http.ResponseWriter, request *http.Request) {
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

	return endpoint
}
