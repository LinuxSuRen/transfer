package main

import (
	"context"
	"fmt"
	"net"
	"os"
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

func newSendCmd() (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "send",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			beginTime := time.Now()
			if len(args) <= 0 {
				fmt.Println("filename is required")
				return
			}

			ip := "0.0.0.0"
			if len(args) >= 2 {
				ip = args[1]
			}

			file := args[0]

			var data []byte
			if data, err = os.ReadFile(file); err != nil {
				return
			}

			fmt.Println("file length", len(data))

			chrunk := 60000
			buffer := make([][]byte, 0)
			index := 0
			for i := 0; i < len(data); index++ {
				j := i + chrunk
				if j > len(data) {
					j = len(data)
				}
				buffer = append(buffer, data[i:j])
				i = j
			}

			var conn net.Conn
			if conn, err = net.Dial("udp", fmt.Sprintf("%s:3000", ip)); err != nil {
				return
			}

			for i := 0; i < len(buffer); i++ {
				// length,filename,count,index
				header := fmt.Sprintf("%s%s%s%s",
					fillContainerWithNumber(len(data), 20),
					fillContainer(file, 100),
					fillContainerWithNumber(len(buffer), 10),
					fillContainerWithNumber(i, 10))
				conn.Write(append([]byte(header), buffer[i]...))

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
						header := fmt.Sprintf("%s%s%s%s",
							fillContainerWithNumber(len(data), 20),
							fillContainer(file, 100),
							fillContainerWithNumber(len(buffer), 10),
							fillContainerWithNumber(index, 10))
						conn.Write(append([]byte(header), buffer[index]...))
					}
				}
			}
			endTime := time.Now()
			fmt.Println("sent over with", endTime.Sub(beginTime).Seconds())

			conn.Close()
			return
		},
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

func confirmOK(conn *net.UDPConn) (ok bool) {
	message := make([]byte, 100)

	if rlen, _, err := conn.ReadFromUDP(message[:]); err == nil {
		ok = string(message[:rlen]) == "confirmed"
	}
	return
}

func sendConfirm(conn *net.UDPConn) (err error) {
	_, err = conn.Write([]byte("confirmed"))
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
						if conn, err := net.Dial("udp", "0.0.0.0:4000"); err == nil {
							_, _ = conn.Write([]byte(fmt.Sprintf("trasfer 0.0.0.0:3000-%d", time.Now().Second())))
							conn.Close()
						}
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
			for {

				f, err := os.CreateTemp(".", "tmp")
				if err != nil {
					panic(err)
				}

				header, err := readHeader(conn)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println("start to receive data from", header.remote)

				buffer := make([][]byte, header.count)
				buffer[header.index] = header.data

				wg := sync.WaitGroup{}
				wg.Add(1)
				go func() {
					defer wg.Done()

					checking := true
					for checking {
						done := true
						for i := 0; i < header.count; i++ {
							if len(buffer[i]) == 0 {
								done = false

								header, err := readHeader(conn)
								if err == nil {
									buffer[header.index] = header.data
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
							if len(buffer[i]) == 0 {
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

				for i := 0; i < header.count; i++ {
					if len(buffer[i]) == 0 {
						fmt.Println("not received", i)
					}
				}

				for i := range buffer {
					f.Write(buffer[i])
				}
				f.Close()

				fmt.Println("writed to file", f.Name())
			}

			return nil
		},
	}
	return
}

type dataHeader struct {
	length   int    // 20 bit
	filename string // 100 bit
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
		if rlen <= 140 {
			err = fmt.Errorf("invalid header format")
			return
		}

		if header.length, err = strconv.Atoi(strings.TrimSpace(string(message[:20]))); err != nil {
			return
		}
		header.filename = strings.TrimSpace(string(message[20:120]))
		if header.count, err = strconv.Atoi(strings.TrimSpace(string(message[120:130]))); err != nil {
			return
		}
		if header.index, err = strconv.Atoi(strings.TrimSpace(string(message[130:140]))); err != nil {
			return
		}

		header.data = message[140:rlen]
	}
	return
}

func writeFile(header dataHeader, data []byte, rlen int) (err error) {
	var f *os.File
	f, err = os.OpenFile(header.filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	_, err = f.Write(data[:rlen])
	if err != nil {
		fmt.Println("failed to write", err)
	} else {
		f.Close()
		fmt.Println("write to file", header.filename)
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
