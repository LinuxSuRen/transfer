package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net"
	"os"
	"regexp"
	"time"
)

func newSendCmd() (cmd *cobra.Command) {
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
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 9981})
	if err != nil {
		return err
	}
	data := make([]byte, 1024)
	_, remoteAddr, err := listener.ReadFromUDP(data)
	if err != nil {
		return err
	}
	o.ip = remoteAddr.IP.String()
	cmd.Println("found target", o.ip)
	return
}

func (o *sendOption) runE(cmd *cobra.Command, args []string) (err error) {
	beginTime := time.Now()
	if len(args) <= 0 {
		cmd.PrintErrln("filename is required")
		return
	}

	file := args[0]

	var f *os.File
	if f, err = os.Open(file); err != nil {
		return
	}

	builder := NewHeaderBuilder(file)
	if err = builder.Build(); err != nil {
		return
	}

	chunk := builder.GetChunk()
	fileSize := builder.GetFileSize()

	cmd.Println("sending chunk size", chunk)
	cmd.Println("file length", fileSize)
	cmd.Println("connect to", o.ip)

	var conn net.Conn
	if conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", o.ip, o.port)); err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	cmd.Println("start to send data")
	reader := bufio.NewReader(f)
	for i := 0; i < builder.GetBufferCount(); i++ {
		buf := make([]byte, builder.GetChunk())
		var n int
		if n, err = reader.Read(buf); err == io.EOF {
			break
		} else if err != nil {
			return
		}

		err = retry(10, func() error {
			// no buffer space available might happen on darwin
			_, err := conn.Write(builder.CreateHeader(i, buf[:n]))
			return err
		})
		if err != nil {
			fmt.Println(err)
		}

		if i == 0 {
			// give more time to init file for the first package
			time.Sleep(time.Second)
		}
	}
	cmd.Println("all the data was sent, try to wait for the missing data")

	checking := true
	go func(checking *bool) {
		cmd.Print("checking")
		for *checking {
			select {
			case <-time.After(time.Second * 5):
				cmd.Print(".")
			}
		}
		cmd.Println()
	}(&checking)

	for checking {
		var index int
		var ok bool
		if index, ok, err = waitingMissing(conn); ok {
			if index == -1 {
				checking = false
			} else {
				if err = send(f, reader, conn, index, chunk, builder); err != nil {
					return err
				}
			}
		} else if err != nil {
			if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
				if conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", o.ip, o.port)); err != nil {
					return
				}
			} else if match, _ := regexp.MatchString(".*i/o timeout.*", err.Error()); match {
				if conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", o.ip, o.port)); err != nil {
					return
				}
			}

			time.Sleep(time.Second * 2)
			if err = send(f, reader, conn, 0, chunk, builder); err != nil {
				return err
			}
		}
	}

	endTime := time.Now()
	cmd.Println("sent over with", endTime.Sub(beginTime).Seconds())
	return
}

func send(f *os.File, reader *bufio.Reader, conn net.Conn, index, chunk int, builder *headerBuilder) (err error) {
	if _, err = f.Seek(int64(index*chunk), 0); err != nil {
		return
	}

	buf := make([]byte, chunk)
	var n int
	if n, err = reader.Read(buf); err != nil {
		return
	}

	_ = retry(10, func() error {
		_, err := conn.Write(builder.CreateHeader(index, buf[:n]))
		return err
	})
	return
}
