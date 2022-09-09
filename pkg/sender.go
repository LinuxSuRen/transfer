package pkg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type UDPSender struct {
	ip   string
	port int

	beginTime time.Time
	endTime   time.Time
}

func NewUDPSender(ip string) *UDPSender {
	return &UDPSender{
		ip:        ip,
		port:      3000,
		beginTime: time.Now(),
	}
}

func (s *UDPSender) WithPort(port int) *UDPSender {
	s.port = port
	return s
}

func (s *UDPSender) Send(msg chan string, file string) (err error) {
	defer close(msg)

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

	msg <- fmt.Sprintf("sending chunk size %d\n", chunk)
	msg <- fmt.Sprintf("file length %d\n", fileSize)
	msg <- fmt.Sprintf("connect to %s\n", s.ip)

	var conn net.Conn
	if conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", s.ip, s.port)); err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	msg <- "start to send data\n"
	reader := bufio.NewReader(f)
	for i := 0; i < builder.GetBufferCount(); i++ {
		buf := make([]byte, builder.GetChunk())
		var n int
		if n, err = reader.Read(buf); err == io.EOF {
			break
		} else if err != nil {
			return
		}

		err = Retry(30, func() error {
			// no buffer space available might happen on darwin
			_, err := conn.Write(builder.CreateHeader(i, buf[:n]))
			return err
		})

		if i == 0 {
			// give more time to init file for the first package
			time.Sleep(time.Second)
		}
	}
	msg <- "all the data was sent, try to wait for the missing data\n"

	mapBuffer := NewSafeMap(0)
	ck := atomic.Bool{}
	ck.Store(true)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		msg <- "checking"

		for index := mapBuffer.GetLowestAndRemove(); ck.Load(); index = mapBuffer.GetLowestAndRemove() {
			if index != nil {
				_ = send(f, reader, conn, *index, chunk, builder)
			} else {
				msg <- "."
				time.Sleep(time.Second * 3)
			}
		}
		msg <- "\n"
	}()

	for ck.Load() {
		var index int
		var ok bool
		if index, ok, err = waitingMissing(conn); ok {
			if index == -1 {
				ck.Store(false)
			} else {
				//fmt.Println("got missing", index)
				mapBuffer.Put(index, "")
			}
		} else if err != nil {
			if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
				time.Sleep(time.Second * 2)

				if conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", s.ip, s.port)); err != nil {
					fmt.Println(err)
					return
				}
				if err = send(f, reader, conn, 0, chunk, builder); err != nil {
					return err
				}
			}
		}
	}

	wg.Wait()
	s.endTime = time.Now()
	msg <- "end"
	return
}

// ConsumedTime returns the consumed time
func (s *UDPSender) ConsumedTime() time.Duration {
	return s.endTime.Sub(s.beginTime)
}

func send(f *os.File, reader *bufio.Reader, conn net.Conn, index, chunk int, builder *HeaderBuilder) (err error) {
	if _, err = f.Seek(int64(index*chunk), 0); err != nil {
		return
	}

	buf := make([]byte, chunk)
	var n int
	if n, err = reader.Read(buf); err != nil {
		return
	}

	err = Retry(30, func() error {
		_, err := conn.Write(builder.CreateHeader(index, buf[:n]))
		return err
	})
	return
}

// waitingMissing read data, returns the missing index.
// Consider it has finished if the index is -1.
func waitingMissing(conn net.Conn) (index int, ok bool, err error) {
	// format: miss0000000012, the index is 12
	message := make([]byte, 14)

	var rlen int
	//if err = conn.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
	//	return
	//}

	if rlen, err = conn.Read(message[:]); err == nil && rlen == 14 {
		index, ok = checkMissing(message)
	}
	return
}

func checkMissing(message []byte) (index int, ok bool) {
	var err error

	if strings.HasPrefix(string(message), "miss") {
		msg := strings.TrimSpace(strings.TrimPrefix(string(message), "miss"))
		if index, err = strconv.Atoi(msg); err == nil {
			ok = true
		}
	} else if strings.HasPrefix(string(message), "done") {
		index = -1
		ok = true
	}
	return
}

// FindWaiters finds the potential package waiters, and notify with a channel
func FindWaiters(ctx context.Context, waiter chan string) {
	go func() {
		for {
			var listener *net.UDPConn
			var err error

			select {
			case <-ctx.Done():
				if listener != nil {
					_ = listener.Close()
				}
				return
			default:
				listener, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 9981})
				if err != nil {
					continue
				}

				data := make([]byte, 1024)
				_, remoteAddr, err := listener.ReadFromUDP(data)
				if err == nil {
					waiter <- remoteAddr.IP.String()
				}
			}
		}
	}()
}
