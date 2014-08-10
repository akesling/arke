package main

import (
    "arke/interchange"
    "arke/endpoint/rest"
    "log"
)

func main() {
    hub := interchange.NewHub()
    endpoint := rest.NewJSONEndpoint(hub.NewClient(), port)

    future, err := endpoint.Start()
    if err != nil {
        log.Fatal(err)
    }

    <-future
}
