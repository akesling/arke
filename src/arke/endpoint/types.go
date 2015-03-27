package endpoint

type Endpoint interface {
	Start(port int) (done <-chan struct{})
	Stop()
}
