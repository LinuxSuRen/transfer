package server

import (
	"context"
	_ "embed"
	"net"
	"net/http"
)

//go:embed index.html
var indexHTML string

// StartHTTPServer starts a simple HTTP server for the index.html file
func StartHTTPServer(ctx context.Context, port chan int) (err error) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(indexHTML))
	})
	var listener net.Listener
	if listener, err = net.Listen("tcp", ":0"); err == nil {
		select {
		case <-ctx.Done():
			_ = listener.Close()
			break
		default:
			port <- listener.Addr().(*net.TCPAddr).Port
			_ = http.Serve(listener, nil)
		}
	}
	return
}

// RsyncStartHTTPServer starts the HTTP server async, and returns the port
func RsyncStartHTTPServer(ctx context.Context) int {
	portChan := make(chan int, 1)
	go func() {
		_ = StartHTTPServer(ctx, portChan)
	}()
	return <-portChan
}
