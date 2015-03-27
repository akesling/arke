/*
Package arke/endpoint contains Arke transport components including server and
client implementations.

# Transports

## HTTP REST Endpoint

### Publication

Resources of the form /topics/bar/baz represent the underlying topic.
POSTing to /topics/bar/baz represents publishing on bar.baz for the hub
represented by foo.com.

### Subscription

Resources of the form "/subscriptions/bar/baz" represent the collection of
subscriptions to the topic bar.baz. POSTing to /subscriptions/bar/baz
represents creation of a new subscription to bar.baz.

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

*/

package endpoint
