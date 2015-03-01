package main

import (
    "arke/interchange"
    "arke/endpoint/rest"
    "arke/endpoint/codex"
    "log"
)

func main() {
    hub := interchange.NewHub()
    endpoint := rest.NewEndpoint(hub.NewClient(), codex.NewJSON(), port)

    future, err := endpoint.Start()
    if err != nil {
        log.Fatal(err)
    }

    <-future
}
