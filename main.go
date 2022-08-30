package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

func NewRoot() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "transfer",
	}

	cmd.AddCommand(newSendCmd(), newWaitCmd())
	return
}

type sendOption struct {
	ip string
}

func (o *sendOption) preRunE(cmd *cobra.Command, args []string) (err error) {
	if len(args) >= 2 {
		o.ip = args[1]
		return
	}

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
		fmt.Println("filename is required")
		return
	}

	file := args[0]

	var f *os.File
	if f, err = os.Open(file); err != nil {
		return
	}

	var fi os.FileInfo
	if fi, err = os.Stat(file); err != nil {
		return
	}

	fileSize := fi.Size()
	fmt.Println("file length", fileSize)

	chrunk := 60000
	if runtime.GOOS == "darwin" {
		chrunk = 500 // default value on darwin is 9216
	}
	fmt.Println("sending chrunk size", chrunk)

	index := 0
	bufferCount := 0
	var i int64
	var j int64
	for i = 0; i < fileSize; index++ {
		j = i + int64(chrunk)
		if j > fileSize {
			j = fileSize
		}
		bufferCount = bufferCount + 1
		i = j
	}

	cmd.Println("connect to", o.ip)
	var conn net.Conn
	if conn, err = net.Dial("udp", fmt.Sprintf("%s:3000", o.ip)); err != nil {
		return
	}

	cmd.Println("start to send data")
	reader := bufio.NewReader(f)
	for i := 0; i < bufferCount; i++ {
		// length,filename,count,index
		header := fmt.Sprintf("%s%s%s%s%s",
			fillContainerWithNumber(int(fileSize), 20),
			fillContainer(path.Base(f.Name()), 100),
			fillContainerWithNumber(chrunk, 10),
			fillContainerWithNumber(bufferCount, 10),
			fillContainerWithNumber(i, 10))

		buf := make([]byte, chrunk)
		var n int
		if n, err = reader.Read(buf); err == io.EOF {
			break
		} else if err != nil {
			return
		}

		retry(10, func() error {
			_, err := conn.Write(append([]byte(header), buf[:n]...))
			return err
		})

		if i == 0 {
			time.Sleep(time.Second)
		}
	}

	checking := true
	for checking {
		if index, ok := waitingMissing(conn); ok {
			if index == -1 {
				checking = false
			} else {
				f.Seek(int64(index*chrunk), 0)

				header := fmt.Sprintf("%s%s%s%s%s",
					fillContainerWithNumber(bufferCount, 20),
					fillContainer(path.Base(f.Name()), 100),
					fillContainerWithNumber(chrunk, 10),
					fillContainerWithNumber(bufferCount, 10),
					fillContainerWithNumber(index, 10))

				buf := make([]byte, chrunk)
				var n int
				if n, err = reader.Read(buf); err != nil {
					return
				}

				retry(10, func() error {
					_, err := conn.Write(append([]byte(header), buf[:n]...))
					return err
				})
			}
		}
	}
	endTime := time.Now()
	fmt.Println("sent over with", endTime.Sub(beginTime).Seconds())

	conn.Close()
	return
}

func retry(count int, callback func() error) (err error) {
	if callback == nil {
		return
	}

	for i := 0; i < count; i++ {
		if err = callback(); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return
}

func newSendCmd() (cmd *cobra.Command) {
	opt := &sendOption{}

	cmd = &cobra.Command{
		Use:     "send",
		PreRunE: opt.preRunE,
		RunE:    opt.runE,
	}
	return
}

// waitingMissing read data, returns the missing index.
// Consider it has finished if the index is -1.
func waitingMissing(conn net.Conn) (index int, ok bool) {
	// format: miss0000000012, the index is 12
	message := make([]byte, 14)

	if rlen, err := conn.Read(message[:]); err == nil && rlen == 14 {
		index, ok = checkMissing(message)
	}
	return
}

func requestMissing(conn *net.UDPConn, index int, remote *net.UDPAddr) (err error) {
	_, err = conn.WriteTo([]byte("miss"+fillContainerWithNumber(index, 10)), remote)
	return
}

func requestDone(conn *net.UDPConn, remote *net.UDPAddr) (err error) {
	for i := 0; i < 3; i++ {
		_, err = conn.WriteTo([]byte("done"+fillContainerWithNumber(0, 10)), remote)

		time.Sleep(time.Second)
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

func fillContainerWithNumber(num, size int) string {
	return fillContainer(fmt.Sprintf("%d", num), size)
}

func fillContainer(txt string, size int) string {
	length := len(txt)
	buf := strings.Builder{}

	prePendCount := size - length
	for i := 0; i < prePendCount; i++ {
		buf.WriteString(" ")
	}
	buf.WriteString(txt)
	return buf.String()
}

func newWaitCmd() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "wait",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, _ := context.WithCancel(context.Background())
			go func(ctx context.Context) {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(3 * time.Second):
						ip := net.ParseIP("192.168.1.255")
						srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
						dstAddr := &net.UDPAddr{IP: ip, Port: 9981}
						conn, err := net.ListenUDP("udp", srcAddr)
						if err != nil {
							return
						}
						conn.WriteToUDP([]byte("hello"), dstAddr)
					}
				}
			}(ctx)

			conn, err := net.ListenUDP("udp", &net.UDPAddr{
				Port: 3000,
				IP:   net.ParseIP("0.0.0.0"),
			})
			if err != nil {
				panic(err)
			}
			defer conn.Close()

			fmt.Printf("server listening %s\n", conn.LocalAddr().String())

			header, err := readHeader(conn)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println("start to receive data from", header.remote)

			f, err := os.OpenFile(header.filename, os.O_WRONLY|os.O_CREATE, 0640)
			if err != nil {
				panic(err)
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

			fmt.Println("wrote to file", f.Name())
			return nil
		},
	}
	return
}

type dataHeader struct {
	length   int    // 20 bit
	filename string // 100 bit
	chrunk   int    // 10 bit
	count    int    // 10 bit
	index    int    // 10 bit
	data     []byte

	remote *net.UDPAddr
}

func readHeader(conn *net.UDPConn) (header dataHeader, err error) {
	message := make([]byte, 65507)
	var rlen int
	rlen, header.remote, err = conn.ReadFromUDP(message[:])
	if err == nil {
		if rlen <= 150 {
			err = fmt.Errorf("invalid header format")
			return
		}

		if header.length, err = strconv.Atoi(strings.TrimSpace(string(message[:20]))); err != nil {
			return
		}
		header.filename = strings.TrimSpace(string(message[20:120]))
		if header.chrunk, err = strconv.Atoi(strings.TrimSpace(string(message[120:130]))); err != nil {
			return
		}
		if header.count, err = strconv.Atoi(strings.TrimSpace(string(message[130:140]))); err != nil {
			return
		}
		if header.index, err = strconv.Atoi(strings.TrimSpace(string(message[140:150]))); err != nil {
			return
		}

		header.data = message[150:rlen]
	}
	return
}

func main() {
	cmd := NewRoot()
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
