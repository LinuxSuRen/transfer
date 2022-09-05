package pkg

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

func TestHeaderBuilder(t *testing.T) {
	file := path.Join(os.TempDir(), "fake")
	err := os.WriteFile(file, []byte("hello"), 0600)
	assert.Nil(t, err)
	defer func() {
		_ = os.RemoveAll(file)
	}()

	builder := NewHeaderBuilder(file)
	assert.Nil(t, builder.Build())

	data := builder.CreateHeader(1, []byte("hello"))
	assert.Equal(t, []byte("                   5                                                                                                fake     60000         1         1hello"), data)
}
