package pkg

import (
	"fmt"
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

func Test_readHeaderFromData(t *testing.T) {
	type args struct {
		data ReceivedData
	}
	tests := []struct {
		name       string
		args       args
		wantHeader dataHeader
		wantErr    assert.ErrorAssertionFunc
	}{{
		name: "not enough lenght",
		args: args{
			data: ReceivedData{
				Data: nil,
			},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
	}, {
		name: "valid header",
		args: args{
			data: ReceivedData{
				Data: readFile("testdata/sample-header.txt"),
			},
		},
		wantHeader: dataHeader{
			length:   12,
			filename: "1.txt",
			chrunk:   1234,
			count:    1,
			index:    1,
			data:     []byte("data"),
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHeader, err := readHeaderFromData(tt.args.data)
			if !tt.wantErr(t, err, fmt.Sprintf("readHeaderFromData(%v)", tt.args.data)) {
				return
			}
			assert.Equalf(t, tt.wantHeader, gotHeader, "readHeaderFromData(%v)", tt.args.data)
		})
	}
}

func readFile(fileName string) (data []byte) {
	data, _ = os.ReadFile(fileName)
	return
}
