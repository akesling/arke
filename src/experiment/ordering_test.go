package experiment

import (
    "fmt"
    "testing"
)

func TestGoroutineSameChannel(t *testing.T) {
    done := make(chan struct{})
    defer close(done)
    pong := make(chan struct{})
    go func() {
        pong<-struct{}{}
        for {
            select {
            case <-pong:
                pong<-struct{}{}
            case <-done:
                break
            }
        }
    }()
    go func() {
        for {
            select {
            case <-pong:
                pong<-struct{}{}
            case <-done:
                break
            }
        }
    }()

    var start, end uint
    start, end = 0, 1e6
    comm := make(chan uint)

    for i := start; i < end; i += 1 {
        go func(sent uint) {
            comm <- sent
        }(i)
    }

    for i := start; i < end; i += 1 {
        if recv := <- comm; i != recv {
            t.Error(fmt.Sprintf("Strict ordering is not preserved. (Sent:%s != Received:%s)", i, recv))
        }
    }
}
