package cmd

import (
	"context"
	"fmt"
	"github.com/linuxsuren/transfer/pkg"
	"github.com/spf13/cobra"
)

func NewSendCmd() (cmd *cobra.Command) {
	opt := &sendOption{}

	cmd = &cobra.Command{
		Use:     "send",
		Short:   "Send data with UDP protocol",
		PreRunE: opt.preRunE,
		RunE:    opt.runE,
	}
	flags := cmd.Flags()
	flags.IntVarP(&opt.port, "port", "p", 3000, "The port to send")
	return
}

type sendOption struct {
	ip   string
	port int
}

func (o *sendOption) preRunE(cmd *cobra.Command, args []string) (err error) {
	if len(args) >= 2 {
		o.ip = args[1]
		return
	}

	cmd.Println("no target ip provided, trying to find it")

	ctx, cancel := context.WithCancel(cmd.Context())

	waiter := make(chan string, 10)
	pkg.FindWaiters(ctx, waiter)

	o.ip = <-waiter
	cancel()
	return
}

func (o *sendOption) runE(cmd *cobra.Command, args []string) (err error) {
	if len(args) <= 0 {
		cmd.PrintErrln("filename is required")
		return
	}

	file := args[0]

	sender := pkg.NewUDPSender(o.ip).WithPort(o.port)
	msg := make(chan string, 10)

	go func() {
		for a := range msg {
			if a == "end" {
				break
			}
			cmd.Print(a)
		}
	}()

	err = sender.Send(msg, file)
	fmt.Printf("sent over in %fs\n", sender.ConsumedTime().Seconds())
	return
}
