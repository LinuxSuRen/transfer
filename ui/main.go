package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/linuxsuren/transfer/pkg"
	"log"
	"net/http"
)

func startHTTPServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(indexHTML))
	})
	_ = http.ListenAndServe(":9999", nil)
}

func main() {
	// Set logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

	go func() {
		startHTTPServer()
	}()

	// Create astilectron
	a, err := astilectron.New(l, astilectron.Options{
		AppName:           "Test",
		BaseDirectoryPath: "example",
	})
	if err != nil {
		l.Fatal(fmt.Errorf("main: creating astilectron failed: %w", err))
	}
	defer a.Close()

	// Handle signals
	a.HandleSignals()

	// Start
	if err = a.Start(); err != nil {
		l.Fatal(fmt.Errorf("main: starting astilectron failed: %w", err))
	}

	// New window
	var w *astilectron.Window
	if w, err = a.NewWindow("http://localhost:9999", &astilectron.WindowOptions{
		Center: astikit.BoolPtr(true),
		Height: astikit.IntPtr(700),
		Width:  astikit.IntPtr(700),
	}); err != nil {
		l.Fatal(fmt.Errorf("main: new window failed: %w", err))
	}
	//w.OpenDevTools()
	w.OnMessage(func(m *astilectron.EventMessage) (v interface{}) {
		fmt.Println("receive message")
		var s string
		err := m.Unmarshal(&s)
		fmt.Println(err)

		fmt.Println(s)
		data := make(map[string]string)
		err = json.Unmarshal([]byte(s), &data)
		fmt.Println(err)

		switch data["cmd"] {
		case "wait":
			waiter := pkg.NewUDPWaiter(3000)
			msg := make(chan string, 10)

			go func() {
				for m := range msg {
					var n = a.NewNotification(&astilectron.NotificationOptions{
						Body:             m,
						HasReply:         astikit.BoolPtr(true),  // Only MacOSX
						ReplyPlaceholder: "type your reply here", // Only MacOSX
						Title:            "Msg",
					})
					// Create notification
					n.Create()
					// Show notification
					n.Show()
				}
			}()
			return waiter.Start(msg)
		case "send":
			file := data["message"]
			if file != "" {
				sender := pkg.NewUDPSender(data["ip"])
				msg := make(chan string, 10)

				go func() {
					for m := range msg {
						var n = a.NewNotification(&astilectron.NotificationOptions{
							Body:             m,
							HasReply:         astikit.BoolPtr(true),  // Only MacOSX
							ReplyPlaceholder: "type your reply here", // Only MacOSX
							Title:            "Msg",
						})
						// Create notification
						n.Create()
						// Show notification
						n.Show()
					}
				}()

				err = sender.Send(msg, file)
			}
		}
		return
	})
	_ = pkg.Broadcast(context.TODO())

	ctx, cancel := context.WithCancel(context.TODO())
	w.On(astilectron.EventNameAppClose, func(e astilectron.Event) (deleteListener bool) {
		cancel()
		return
	})
	waiter := make(chan string, 10)
	pkg.FindWaiters(ctx, waiter)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ip := <-waiter:
				_ = w.SendMessage(ip, func(m *astilectron.EventMessage) {
				})
			}
		}
	}()

	// Create windows
	if err = w.Create(); err != nil {
		l.Fatal(fmt.Errorf("main: creating window failed: %w", err))
	}

	// Blocking pattern
	a.Wait()
}

//go:embed index.html
var indexHTML string
