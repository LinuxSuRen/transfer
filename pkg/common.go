package pkg

import (
	"fmt"
	"strings"
	"time"
)

// Retry run the callback for the specific times
func Retry(count int, callback func() error) (err error) {
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
