package main

import (
	"github.com/akesling/arke/codex"
	"github.com/akesling/arke/endpoint/httprest"
	"github.com/akesling/arke/interchange"
	"golang.org/x/net/context"
	"log"
)

func main() {
	hub_ctx, cancel_hub := context.WithCancel(context.Background())
	defer cancel_hub()

	hub := interchange.NewHub(hub_ctx, cancel_hub)
	endpoint := httprest.NewHTTPRestEndpoint(interchange.NewClient(hub), codex.JSONCodex{})

	port := 8080
	endpoint.SetPort(port)
	endpoint_done, err := endpoint.Start()
	if err != nil {
		log.Fatal(err)
	}

	<-endpoint_done
}
