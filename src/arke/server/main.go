package main

import (
	"arke/endpoint/codex"
	"arke/endpoint/httprest"
	"arke/interchange"
	"code.google.com/p/go.net/context"
	"log"
)

func main() {
	hub_ctx, cancel_hub := context.WithCancel(context.Background())
	hub := interchange.NewHub(hub_ctx)
	endpoint := rest.NewEndpoint(hub.NewClient(), codex.NewJSON())

	port := "8080"
	endpoint_done, err := endpoint.Start(port)
	if err != nil {
		log.Fatal(err)
	}

	<-endpoint_done
	cancel_hub()
}
