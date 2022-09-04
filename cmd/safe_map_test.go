package cmd

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestSafeMap(t *testing.T) {
	safeMap := NewSafeMap(1000)
	assert.NotNil(t, safeMap)
	assert.Equal(t, 1000, safeMap.Size())

	wg := sync.WaitGroup{}
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()

			safeMap.Remove(k)
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 500, safeMap.Size())

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = safeMap.GetAndRemove()
		}()
	}
	wg.Wait()
	assert.Equal(t, 0, safeMap.Size())
}
