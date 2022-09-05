package pkg

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestQueue(t *testing.T) {
	queue := &Queue{}

	val := queue.Pop()
	assert.Nil(t, val)

	queue.Push(1)
	val = queue.Pop()
	assert.NotNil(t, val)
	assert.Equal(t, 1, *val)

	// multi-threads case
	wg := sync.WaitGroup{}
	threadCounts := 1000
	for i := 0; i < threadCounts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queue.Push(1)
		}()
	}
	wg.Wait()
	assert.Equal(t, threadCounts, queue.Size())

	// Pop case
	for i := 0; i < threadCounts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queue.Pop()
		}()
	}
	wg.Wait()
	assert.Equal(t, 0, queue.Size())
}
