package main

import (
	"fmt"
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
		// mainly do this on the darwin
		time.Sleep(500 * time.Millisecond)
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
