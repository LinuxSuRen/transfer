package pkg

import (
	"context"
	"net"
	"time"
)

// Broadcast sends the broadcast message to all the potential ip addresses
func Broadcast(ctx context.Context) (err error) {
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
		}(ctx, ip)
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
