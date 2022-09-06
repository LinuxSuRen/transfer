package cmd

import (
	"github.com/linuxsuren/transfer/pkg"
	"github.com/spf13/cobra"
)

type waitOption struct {
	port   int
	listen string
}

func (o *waitOption) preRunE(cmd *cobra.Command, _ []string) (err error) {
	err = pkg.Broadcast(cmd.Context())
	return
}

func (o *waitOption) runE(cmd *cobra.Command, args []string) error {
	waiter := pkg.NewUDPWaiter(o.port).ListenAddress(o.listen)
	msg := make(chan string, 10)

	go func() {
		for a := range msg {
			cmd.Println(a)
		}
	}()
	return waiter.Start(msg)
}

func NewWaitCmd() (cmd *cobra.Command) {
	opt := &waitOption{}
	cmd = &cobra.Command{
		Use:     "wait",
		Short:   "Wait the data from a UDP protocol",
		PreRunE: opt.preRunE,
		RunE:    opt.runE,
	}
	flags := cmd.Flags()
	flags.IntVarP(&opt.port, "port", "p", 3000, "The port to listen")
	flags.StringVarP(&opt.listen, "listen", "l", "0.0.0.0", "The address that want to listen")
	return
}
