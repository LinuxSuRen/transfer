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
	udpAddress := &net.UDPAddr{
		Port: o.port,
		IP:   net.ParseIP(o.listen),
	}

	conn, err := net.ListenUDP("udp", udpAddress)
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
	defer func() {
		_ = f.Close()
	}()

	if _, err = f.Write(make([]byte, header.length)); err != nil {
		err = fmt.Errorf("failed to init file, %v", err)
		return err
	}

	mapBuffer := NewSafeMap(header.count)
	go func() {
		if _, err := f.WriteAt(header.data, int64(header.chrunk*header.index)); err == nil {
			mapBuffer.Remove(header.index)
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		//startedMissingThread := false
		for size := mapBuffer.Size(); size > 0; size = mapBuffer.Size() {
			header, err = readHeader(conn)
			if err == nil {
				go func(header dataHeader) {
					_, err = f.WriteAt(header.data, int64(header.chrunk*header.index))
					if err == nil {
						mapBuffer.Remove(header.index)
					} else {
						fmt.Println(err)
					}
				}(header)
			}
		}
	}()

	// check the buffer and start the missing thread
	lastCount := 0
	time.Sleep(time.Microsecond * time.Duration(lastCount) * 50)
	for lastCount != mapBuffer.Size() {
		lastCount = mapBuffer.Size()
		time.Sleep(time.Second * 5)
	}
	sendWaitingMissingRequest(&wg, &header, mapBuffer, conn)

	wg.Wait()
	cmd.Println("wrote to file", f.Name())
	return nil
}

func sendWaitingMissingRequest(wg *sync.WaitGroup, header *dataHeader, buffer *SafeMap, conn *net.UDPConn) {
	wg.Add(1)
	go func() {
		defer func() {
			_ = conn.Close()
			wg.Done()
		}()

		for buffer.Size() > 0 {
			missing := buffer.GetKeys()
			//fmt.Println("missing", len(missing))
			for _, i := range missing {
				err := requestMissing(conn, i, header.remote)
				if err != nil {
					fmt.Println(err)
				}
			}
			time.Sleep(time.Second)
		}

		for err := requestDone(conn, header.remote); err != nil; {
		}
		fmt.Println("done with checking")
	}()
}

func requestDone(conn *net.UDPConn, remote *net.UDPAddr) (err error) {
	for i := 0; i < 3; i++ {
		_, err = conn.WriteTo([]byte("done"+fillContainerWithNumber(0, 10)), remote)

		time.Sleep(time.Second)
	}
	return
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
