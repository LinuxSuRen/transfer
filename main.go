package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
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

// waitingMissing read data, returns the missing index.
// Consider it has finished if the index is -1.
func waitingMissing(conn net.Conn) (index int, ok bool, err error) {
	// format: miss0000000012, the index is 12
	message := make([]byte, 14)

	var rlen int
	if err = conn.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
		return
	}

	if rlen, err = conn.Read(message[:]); err == nil && rlen == 14 {
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

func main() {
	cmd := NewRoot()
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
