package main

import (
	"encoding/json"
	"fmt"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	cmd2 "github.com/linuxsuren/transfer/cmd"
	"log"
)

func main() {
	// Set logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

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
	if w, err = a.NewWindow("index.html", &astilectron.WindowOptions{
		Center: astikit.BoolPtr(true),
		Height: astikit.IntPtr(700),
		Width:  astikit.IntPtr(700),
	}); err != nil {
		l.Fatal(fmt.Errorf("main: new window failed: %w", err))
	}

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
			cmd := cmd2.NewWaitCmd()
			cmd.Execute()
		case "send":
			file := data["message"]
			if file != "" {
				cmd := cmd2.NewSendCmd()
				cmd.SetArgs([]string{file})
				cmd.Execute()
			}
		}
		return
	})

	// Create windows
	if err = w.Create(); err != nil {
		l.Fatal(fmt.Errorf("main: creating window failed: %w", err))
	}

	// Blocking pattern
	a.Wait()
}
