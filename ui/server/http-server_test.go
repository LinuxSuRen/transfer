package server

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

func Test_startHTTPServer(t *testing.T) {
	portChan := make(chan int, 1)
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		err := StartHTTPServer(ctx, portChan)
		assert.Nil(t, err)
	}()
	select {
	case port := <-portChan:
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		assert.Nil(t, err)

		var body []byte
		body, err = io.ReadAll(resp.Body)
		assert.Nil(t, err)
		assert.Equal(t, indexHTML, string(body))

		assert.NotEqual(t, 0, port)
		cancel()
	}
}
