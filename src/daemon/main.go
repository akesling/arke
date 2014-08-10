package main

import (
    "arke/interchange"
    "arke/endpoint/rest"
)

func main() {
    hub := interchange.NewHub()
    endpoint := rest.NewJSONEndpoint(hub.NewClient(), port)
    <-endpoint.Start()
}
