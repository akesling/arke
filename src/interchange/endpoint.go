package interchange

// An Endpoint provides remote services access to an Arke hub.
interface Endpoint {
    Start() (completion chan<-, error)
}
