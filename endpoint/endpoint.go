/*
Package arke/endpoint contains Arke transport components including server and
client implementations.
*/
package endpoint

// Endpoints are wrappers for Arke Hubs which communicate with external clients.
type Endpoint interface {
	// Start will either provide a done-channel or an endpoint-specific error.
	// Upon occurrence of a runtime error, that error will be store for
	// retrieval via GetError().
	Start() (done <-chan struct{}, err error)

	// Stop halts this endpoint, _not_ any underlying hub.
	// It is idempotent and will not interfere with any stored error.
	// TODO(akesling): Implement stop.
	//	Stop()

	// GetError returns runtime errors for this Endpoint.
	// The error stored will be nil if no error has occurred and will be reset
	// upon call to Start().
	GetError() (err error)
}

// PortEndpoint communicates to hub clients via a particular system port.
type PortEndpoint interface {
	Endpoint

	// SetPort sets the port to use for this endpoint.
	// It does not verify whether this port is available.  Any errors regarding
	// port collision will be returned by Start().
	//
	// The bound port is only used upon call to Start(), thus switching ports
	// requires Stop()+SetPort(foo)+Start().
	SetPort(port int) (err error)
}
