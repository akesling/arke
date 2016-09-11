package main

import (
	"code.google.com/p/go.net/context"
	"github.com/akesling/arke/endpoint/codex"
	"github.com/akesling/arke/endpoint/httprest"
	"github.com/akesling/arke/interchange"
	"log"
)

func main() {
	hub_ctx, cancel_hub := context.WithCancel(context.Background())
	defer cancel_hub()

	hub := interchange.NewHub(hub_ctx)
	endpoint := httprest.NewEndpoint(hub.NewClient(), codex.NewJSON())

	port := "8080"
	endpoint_done, err := endpoint.Start(port)
	if err != nil {
		log.Fatal(err)
	}

	<-endpoint_done
}
