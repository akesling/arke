/*

endpoint/httprest is an HTTP REST endpoint for the Arke pub/sub system.

All bodies in HTTP requests discussed below are encoded in the fashion
configured by the endpoint's codex.

Publication

Resources of the form /topics/bar/baz represent the underlying topic.
POSTing to /topics/bar/baz represents publishing on bar.baz for the hub
represented by foo.com.

Upon success, the status code will be 201 (Creation).

Upon failure, the status code will be an appropriate 4xx dependent on the error,
with the response body being a single "error_message" field (in the encoding
format specified by the endpoint) containing a human readable error string.

Subscription

Resources of the form "/subscriptions/bar/baz" represent the collection of
subscriptions to the topic bar.baz. POSTing to /subscriptions/bar/baz
represents creation of a new subscription to bar.baz.
The following fields are required:

 - address: The return address (e.g. http://foo.bar/baz) of the subscriber at
            which to receive messages and subscription updates.
 - lease_duration: The desired amount of time for which this subscription should
                   last, encoded as integer seconds.

Upon success, the status code will be 201 (Creation) with a body containing the
field "lease_duration" in integer seconds.

Upon failure, the status code will be an appropriate 4xx dependent on the error,
with the response body being a single "error_message" field containing a human
readable error string if one exists.

The HTTP REST Endpoint only supports asynchronous HTTP subscriptions.
This means that a subscription request must include an HTTP endpoint at which
the hub may push updates on the subscription.  The endpoint provided by the
subscriber semantically represents the collection of publications on the
subscriber; thus it must be configured to accept: POST requests (representing new
additions to the collection of messages produced to the given subscriber) and
PUT requests (representing modification of the subscription state itself).

A PUT request will only occur once and will only bear information associated
with subscriber lease expiration (which may occur in a number of situations,
including the lease-time expiring and the hub shutting down).

POSTed updates on the subscriber endpoint will contain the message in its body
and will continue POSTing the same message until the HTTP request is complete...
ignoring all status codes returned (e.g. a 500 response will be considered a
successful delivery).
*/
package httprest

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/akesling/arke/codex"
	"github.com/akesling/arke/endpoint"
	"github.com/akesling/arke/interchange"
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxLeaseDuration time.Duration = time.Duration(5) * time.Minute
)

type httprest struct {
	hub     interchange.Client
	port    int
	mux     *http.ServeMux
	server  *http.Server
	codex   codex.Codex
	logger  *log.Logger
	lastErr error
}

func httpPut(c *http.Client, url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
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

func (h *httprest) Subscribe(subscriberURL, topic string, lease time.Duration) error {
	// TODO(akesling): Add a GET call to the subscriberURL to verify validity
	// before adding a hub subscription.
	var messages <-chan interchange.Message
	messages, err := h.hub.Subscribe(subscriberURL, topic, lease)

	if err != nil {
		// TODO(akesling): Log this error condition and return the error
		// wrapped appropriately.
		return err
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
				var encoded []byte
				if m.Encoding == nil {
					encoded, err = h.codex.Marshal(m.Body)
				} else {
					bodyBytes, ok := m.Body.([]byte)
					if !ok {
						// TODO(akesling): Log this error.
						// Perhaps also account for publishers that incorrectly
						// set m.Encoding.
						continue
					}
					encoded, err = h.codex.Transmarshal(m.Encoding, bodyBytes)
				}

				if err != nil {
					// TODO(akesling): Log this error.
					continue
				}

				_, err := client.Post(subscriberURL, h.codex.MIME(), bytes.NewReader(encoded))
				if err != nil {
					// TODO(akesling): Log this error.
					// Perhaps also account for high-error-rate subscribers.
					// TODO(akesling): Retry on appropriate errors.
					continue
				}
			case <-time.After(lease):
				encoded, err := h.codex.Marshal(map[string]string{"status": "lease expired"})
				if err != nil {
					// TODO(akesling): Log this error.
				}

				_, err = httpPut(client, subscriberURL, h.codex.MIME(), bytes.NewReader(encoded))
				if err != nil {
					// TODO(akesling): Log this error.
					// Perhaps also account for high-error-rate subscribers.
					// TODO(akesling): Retry on appropriate errors.
					continue
				}
				break SubscribeLoop
				// TODO(akesling): Handle server Stop() event.
				// case <-done:
				// break
			}
		}
	}()

	return nil
}

func constrainLease(requestedLease time.Duration) time.Duration {
	switch {
	case requestedLease > maxLeaseDuration:
		return maxLeaseDuration
	case requestedLease < time.Duration(0):
		return time.Duration(0)
	default:
		return requestedLease
	}
}

func decodeTopicURLPath(path string) (topic string, err error) {
	tokens := strings.Split(path, "/")
	parts := make([]string, 0, len(tokens))
	for i := range tokens {
		if tokens[i] == "" {
			continue
		}

		piece, err := url.QueryUnescape(tokens[i])
		if err != nil {
			// TODO(akesling): Improve quality of error message.
			return "", errors.New("URL Path failed topic decoding")
		}

		parts = append(parts, piece)
	}
	topic = strings.Join(parts, ".")

	if topic == "" {
		return ".", nil
	}
	return topic, nil
}

func handleTopicRequest(endpoint *httprest, codex codex.Codex) func(rw http.ResponseWriter, request *http.Request) {
	return func(rw http.ResponseWriter, request *http.Request) {
		topic, err := decodeTopicURLPath(request.URL.Opaque)
		if err != nil {
			endpoint.logger.Printf("Failed to decode provided topic from URL (%q) with error: %q", request.URL.Opaque, err)
			rw.WriteHeader(http.StatusBadRequest)
			// TODO(akesling): Write out a human-readable error message.
			return
		}

		bodyBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			endpoint.logger.Printf("Failed read request body with error: %q", err)
			rw.WriteHeader(http.StatusBadRequest)
			// TODO(akesling): Write out a human-readable error message.
			return
		}
		var message interface{}
		err = codex.Unmarshal(bodyBytes, &message)
		if err != nil {
			endpoint.logger.Printf("Failed to unmarshal request body using codex with MIME %q resulting in error: %q", codex.MIME(), err)
			rw.WriteHeader(http.StatusBadRequest)
			// TODO(akesling): Write out a human-readable error message.
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
				encoded, err := codex.Marshal(map[string]string{"error_message": err.Error()})
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
	}
}

func handleSubscriptionRequest(endpoint *httprest, codex codex.Codex) func(rw http.ResponseWriter, request *http.Request) {
	// TODO(akesling): In all errors, return more valuable human-readable
	// error in the body.

	return func(rw http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case "POST":
			topic, err := decodeTopicURLPath(request.URL.Opaque)
			if err != nil {
				// TODO(akesling): log error
				rw.WriteHeader(http.StatusBadRequest)
				// TODO(akesling): Write out a human-readable error message.
				return
			}

			bodyBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				// TODO(akesling): log error
				rw.WriteHeader(http.StatusBadRequest)
				// TODO(akesling): Write out a human-readable error message.
				return
			}

			var requestFields map[string]string
			err = codex.Unmarshal(bodyBytes, requestFields)
			if err != nil {
				endpoint.logger.Printf("Failed to unmarshal request body as map[string]string using codex with MIME %q resulting in error: %q", codex.MIME(), err)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			var subscriberURL string
			subscriberURL, ok := requestFields["address"]
			if !ok {
				// TODO(akesling): Log error
				rw.WriteHeader(http.StatusBadRequest)
				// TODO(akesling): Write out a human-readable error message.
				return
			}

			var requestedLeaseString string
			requestedLeaseString, ok = requestFields["lease_duration"]
			if !ok {
				// TODO(akesling): Log error
				rw.WriteHeader(http.StatusBadRequest)
				// TODO(akesling): Write out a human-readable error message.
				return
			}

			requestedLease, err := strconv.ParseInt(requestedLeaseString, 10, 64)
			if err != nil {
				// TODO(akesling): Log error
				rw.WriteHeader(http.StatusBadRequest)
				// TODO(akesling): Write out a human-readable error message.
				return
			}
			actualLease := constrainLease(
				time.Duration(requestedLease) * time.Second)

			// Leases with the nil duration shouldn't _do_ anything.
			if actualLease == 0 {
				rw.WriteHeader(http.StatusCreated)
				encoded, err := codex.Marshal(map[string]string{"lease_duration": "0"})
				if err != nil {
					// TODO(akesling): Log error
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.Write(encoded)
				return
			}

			err = endpoint.Subscribe(subscriberURL, topic, actualLease)
			if err != nil {
				rw.WriteHeader(http.StatusForbidden)
				encoded, err := codex.Marshal(map[string]string{"error_message": err.Error()})
				if err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				rw.Write(encoded)
				return
			}
			encoded, err := codex.Marshal(
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
	}
}

func NewHTTPRestEndpoint(hub interchange.Client, codex codex.Codex) endpoint.PortEndpoint {
	newMux := http.NewServeMux()
	endpointLogger, err := syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_USER, log.Lshortfile)
	if err != nil {
		log.Panic("HTTPRestEndpoint logger could not be constructed.")
	}
	newEndpoint := &httprest{hub: hub, mux: newMux, codex: codex, logger: endpointLogger}
	newEndpoint.logger.Print("Foo!")

	newMux.HandleFunc("subscriptions", handleSubscriptionRequest(newEndpoint, codex))
	newMux.HandleFunc("topics", handleTopicRequest(newEndpoint, codex))

	return endpoint.PortEndpoint(newEndpoint)
}
