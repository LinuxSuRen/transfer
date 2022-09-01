package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"net"
	"os"
	"sync"
	"time"
)

type waitOption struct {
	port   int
	listen string
}

func (o *waitOption) preRunE(cmd *cobra.Command, args []string) (err error) {
	var ifaces []net.Interface
	if ifaces, err = net.Interfaces(); err != nil {
		return
	}

	var addrs []net.Addr
	var allIPs []net.IP
	for _, i := range ifaces {
		if addrs, err = i.Addrs(); err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			}

			if ip == nil || ip.IsLinkLocalUnicast() || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			allIPs = append(allIPs, ip)
		}
	}

	for _, ip := range allIPs {
		go func(ctx context.Context, ip net.IP) {
			broadcast(ctx, ip)
		}(cmd.Context(), ip)
	}
	return
}

func broadcast(ctx context.Context, ip net.IP) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			ip.To4()[3] = 255
			srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
			dstAddr := &net.UDPAddr{IP: ip, Port: 9981}
			conn, err := net.ListenUDP("udp", srcAddr)
			if err != nil {
				return
			}
			_, _ = conn.WriteToUDP([]byte("hello"), dstAddr)
		}
	}
}

func (o *waitOption) runE(cmd *cobra.Command, args []string) error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: o.port,
		IP:   net.ParseIP(o.listen),
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	cmd.Printf("server listening %s\n", conn.LocalAddr().String())

	header, err := readHeader(conn)
	if err != nil {
		cmd.Println(err)
		return err
	}
	cmd.Println("start to receive data from", header.remote)

	f, err := os.OpenFile(header.filename, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}

	if _, err = f.Write(make([]byte, header.length)); err != nil {
		err = fmt.Errorf("failed to init file, %v", err)
		return err
	}

	buffer := make([]byte, header.count)
	buffer[header.index] = 1
	f.WriteAt(header.data, int64(header.chrunk*header.index))

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		checking := true
		for checking {
			done := true
			for i := 0; i < header.count; i++ {
				if buffer[i] != 1 {
					done = false

					header, err := readHeader(conn)
					if err == nil {
						_, err = f.WriteAt(header.data, int64(header.chrunk*header.index))
						if err == nil {
							buffer[header.index] = 1
						}
					}
				}
			}
			if done {
				checking = false
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		checking := true
		for checking {
			done := true
			for i := 0; i < header.count; i++ {
				if buffer[i] != 1 {
					done = false
					requestMissing(conn, i, header.remote)
					time.Sleep(time.Millisecond * 10)
					break
				}
			}

			if done {
				requestDone(conn, header.remote)
				checking = false
			}
		}
		fmt.Println("done with checking")
	}()

	wg.Wait()
	f.Close()

	cmd.Println("wrote to file", f.Name())
	return nil
}

func newWaitCmd() (cmd *cobra.Command) {
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
