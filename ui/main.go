package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/linuxsuren/transfer/pkg"
	"github.com/linuxsuren/transfer/ui/server"
	"log"
)

func main() {
	// Set logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())
	portChan := make(chan int, 1)
	ctx, cancel := context.WithCancel(context.TODO())

	go func() {
		if err := server.StartHTTPServer(ctx, portChan); err != nil {
			fmt.Println(err)
		}
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
	if w, err = a.NewWindow(fmt.Sprintf("http://localhost:%d", <-portChan), &astilectron.WindowOptions{
		Center: astikit.BoolPtr(true),
		Height: astikit.IntPtr(500),
		Width:  astikit.IntPtr(700),
	}); err != nil {
		l.Fatal(fmt.Errorf("main: new window failed: %w", err))
	}
	//w.OpenDevTools()
	w.OnMessage(func(m *astilectron.EventMessage) (v interface{}) {
		var s string
		if err := m.Unmarshal(&s); err != nil {
			return
		}

		data := make(map[string]string)
		err = json.Unmarshal([]byte(s), &data)

		switch data["cmd"] {
		case "wait":
			waiter := pkg.NewUDPWaiter(3000)
			msg := make(chan string, 10)

			go func() {
				for m := range msg {
					_ = sendMsg("log", m, w)
				}
			}()
			return waiter.Start(msg)
		case "stopWait":
			fmt.Println("not support stop wait")
		case "send":
			file := data["message"]
			if file != "" {
				sender := pkg.NewUDPSender(data["ip"])
				msg := make(chan string, 10)

				go func() {
					for m := range msg {
						if m == "end" {
							break
						}
						_ = sendMsg("log", m, w)
					}
				}()

				err = sender.Send(msg, file)
				_ = sendMsg("log", fmt.Sprintf("sent over in %fs\n", sender.ConsumedTime().Seconds()), w)
			}
		}
		return
	})
	_ = pkg.Broadcast(context.TODO())

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
				_ = sendMsg("ip", ip, w)
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

func sendMsg(key, value string, w *astilectron.Window) error {
	return w.SendMessage(map[string]string{
		"key":   key,
		"value": value,
	})
}
