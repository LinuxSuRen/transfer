package main

import (
	cmd2 "github.com/linuxsuren/transfer/cmd"
	"github.com/spf13/cobra"
)

func NewRoot() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "transfer",
	}

	cmd.AddCommand(cmd2.NewSendCmd(), cmd2.NewWaitCmd())
	return
}

func main() {
	cmd := NewRoot()
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
