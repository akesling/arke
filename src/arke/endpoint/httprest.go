package endpoint

import (
	"arke/codex"
	"arke/interchange"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxLeaseDuration = time.Duration(5) * time.Minute
)

type httprest struct {
	hub     *interchange.Client
	port    int
	mux     *http.ServeMux
	server  *http.Server
	codex   codex.Codex
	lastErr error
}

func (h *httprest) SetPort(port int) (err error) {
	h.port = port
	return nil
}

func (h *httprest) Start() (done <-chan struct{}, err error) {
	h.lastErr = nil

	// TODO(akesling): start the http server and hold a handle to stop it later.
	if h.port == 0 {
		h.lastErr = errors.New("Port has not been set. Please call SetPort() with a valid port number.")
		return nil, h.lastErr
	}

	// TODO(akesling): if the server is already running, return an appropriate
	// error here.

	// TODO(akesling): implement AsyncServer to use http.Hijacker.Hijack to stop
	// the underlying http.Server goroutine.

	portString := fmt.Sprintf(":%d", h.port)
	h.server = &http.Server{
		Addr:           portString,
		Handler:        h.mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err = h.server.ListenAndServe()
	if err != nil {
		h.lastErr = err
		return nil, h.lastErr
	}
	return nil, nil
}

/*
// TODO(akesling): Implement stop.
func (h *httprest) Stop() {
	h.server.Stop()
}
*/

func (h *httprest) Publish(topic string, message interchange.Message) error {
	return h.hub.Publish(topic, message)
}

func (h *httprest) GetError() (err error) {
	return h.lastErr
}

func (h *httprest) Subscribe(subscriberURL, topic string, lease time.Duration) (<-chan interchange.Message, error) {
	// TODO(akesling): Add a GET call to the subscriberURL to verify validity
	// before adding a hub subscription.
	messages, err := h.hub.Subscribe(subscriberURL, topic, lease)

	if err != nil {
		// TODO(akesling): Log this error condition and return the error
		// wrapped appropriately.
		return nil, err
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

				resp, err := client.Post(subscriberURL, h.codex.MIME(), bytes.NewReader(encoded))
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

	return messages, nil
}

func constrainLease(requestedLease time.Duration) time.Duration {
	switch requestedLease {
	case requestedLease > maxLeaseDuration:
		return maxLeaseDuration
	case requestedLease < time.Duration(0):
		return time.Duration(0)
	default:
		return requestedLease
	}
}

func decodeTopicURLPath(path string) (topic string, err error) {
	tokens := strings.Split("/", path)
	for i := range tokens {
		tokens[i], err = url.QueryUnescape(tokens[i])
		if err != nil {
			// TODO(akesling): Improve quality of error message.
			return "", errors.New("URL Path failed topic decoding")
		}
	}
	topic = strings.Join(tokens, ".")
	return topic, nil
}

func NewEndpoint(hub *interchange.Client, codex codex.Codex) *PortEndpoint {
	newMux := http.NewServeMux()
	endpoint := &httprest{hub: hub, mux: newMux, codex: codex}

	newMux.HandleFunc("subscriptions", func(rw http.ResponseWriter, request *http.Request) {
		// TODO(akesling): In all errors, return more valuable human-readable
		// error in the body.

		switch request.Method {
		case "POST":
			topic, err := decodeTopicURLPath(request.Opaque)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			bodyBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			requestObject, err := codex.Decode(bodyBytes)
			requestFields, ok := requestObject.(map[string]string)
			if err != nil || !ok {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			var subscriberURL string
			if subscriberURL, ok := requestFields["address"]; !ok {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			var requestedLeaseString string
			if requestedLeaseString, ok := requestFields["lease_duration"]; !ok {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			requestedLease, err := strconv.ParseInt(requestedLeaseString, 10, 64)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			actualLease := constrainLease(
				time.Duration(requestedLease) * time.Second)

			// Leases with the nil duration shouldn't _do_ anything.
			if actualLease == 0 {
				rw.WriteHeader(http.StatusCreated)
				encoded, err := codex.Encode(map[string]string{"lease_duration": "0"})
				if err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.Write(encoded)
				return
			}

			messages, err := endpoint.Subscribe(subscriberURL, topic, actualLease)
			if err != nil {
				rw.WriteHeader(http.StatusForbidden)
				encoded, err := codex.Encode(map[string]string{"error_message": err.Error()})
				if err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.Write(encoded)
				return
			}
			encoded, err := codex.Encode(
				map[string]string{
					"lease_duration": fmt.Sprintf("%d", actualLease.Seconds()),
				})
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			rw.WriteHeader(http.StatusCreated)
			rw.Write(encoded)
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
			// TODO(akesling): include the appropriate Allow header.
		}
	})

	newMux.HandleFunc("topics", func(rw http.ResponseWriter, request *http.Request) {
		topic, err := decodeTopicURLPath(request.Opaque)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		bodyBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		message, err := codex.Decode(bodyBytes)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		switch request.Method {
		case "POST":
			err := endpoint.Publish(
				topic,
				interchange.Message{
					Encoding: codex,
					Source:   request.RemoteAddr,
					Body:     message,
				})
			if err != nil {
				encoded, err := codex.Encode(map[string]string{"error_message": err.Error()})
				if err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.WriteHeader(http.StatusForbidden)
				rw.Write(encoded)
				return
			}

			rw.WriteHeader(http.StatusCreated)
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
			// TODO(akesling): include the appropriate Allow header.
		}
	})

	temp := PortEndpoint(endpoint)
	return &temp
}
