package main

import (
	"context"
	"fmt"
	"github.com/linuxsuren/transfer/ui/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.TODO())

	port := server.RsyncStartHTTPServer(ctx)
	fmt.Println("started with port:", port)
	for {
		switch <-s {
		case syscall.SIGKILL:
			cancel()
		}
	}
}
