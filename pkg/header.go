package pkg

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
)

type dataHeader struct {
	length   int    // 20 bit
	filename string // 100 bit
	chrunk   int    // 10 bit
	count    int    // 10 bit
	index    int    // 10 bit
	data     []byte

	remote *net.UDPAddr
}

func readHeaderFromData(data ReceivedData) (header dataHeader, err error) {
	message := data.Data
	rlen := len(message)
	if rlen <= 150 {
		err = fmt.Errorf("invalid header format, message length should bigger than 150, current is %d", rlen)
		return
	}

	length := string(message[:20])
	header.filename = strings.TrimSpace(string(message[20:120]))
	chrunk := string(message[120:130])
	count := string(message[130:140])
	index := string(message[140:150])

	if header.length, err = strconv.Atoi(strings.TrimSpace(length)); err != nil {
		err = fmt.Errorf("invalid length: '%s'", string(message[:20]))
		return
	}
	if header.chrunk, err = strconv.Atoi(strings.TrimSpace(chrunk)); err != nil {
		err = fmt.Errorf("invalid chrunk: '%s'", string(message[120:130]))
		return
	}
	if header.count, err = strconv.Atoi(strings.TrimSpace(count)); err != nil {
		return
	}
	if header.index, err = strconv.Atoi(strings.TrimSpace(index)); err != nil {
		return
	}

	header.remote = data.Remote
	header.data = message[150:rlen]
	return
}

func readHeader(conn *net.UDPConn) (header dataHeader, err error) {
	message := make([]byte, 65507)
	var rlen int
	data := ReceivedData{}
	if rlen, data.Remote, err = conn.ReadFromUDP(message[:]); err != nil {
		return
	}
	data.Data = message[:rlen]
	return readHeaderFromData(data)
}

type HeaderBuilder struct {
	file string

	filename    string
	fileSize    int64
	chunk       int
	bufferCount int
}

// NewHeaderBuilder creates an instance of the HeaderBuilder
func NewHeaderBuilder(file string) *HeaderBuilder {
	return &HeaderBuilder{
		file: file,
	}
}

func (h *HeaderBuilder) Build() (err error) {
	var fi os.FileInfo
	if fi, err = os.Stat(h.file); err != nil {
		return
	}

	h.fileSize = fi.Size()
	h.filename = path.Base(fi.Name())

	switch runtime.GOOS {
	case "darwin":
		h.chunk = 9000 // default value on darwin is 9216
	default:
		h.chunk = 60000
	}

	var (
		index int
		i, j  int64
	)
	for i = 0; i < h.fileSize; index++ {
		j = i + int64(h.chunk)
		if j > h.fileSize {
			j = h.fileSize
		}
		h.bufferCount = h.bufferCount + 1
		i = j
	}
	return
}

// CreateHeader creates the header with index
func (h *HeaderBuilder) CreateHeader(index int, data []byte) []byte {
	// length,filename,count,index
	header := fmt.Sprintf("%s%s%s%s%s",
		fillContainerWithNumber(int(h.GetFileSize()), 20),
		fillContainer(h.GetFilename(), 100),
		fillContainerWithNumber(h.GetChunk(), 10),
		fillContainerWithNumber(h.GetBufferCount(), 10),
		fillContainerWithNumber(index, 10))
	return append([]byte(header), data...)
}

// GetChunk returns the chunk size
func (h *HeaderBuilder) GetChunk() int {
	return h.chunk
}

// GetBufferCount returns the buffer count
func (h *HeaderBuilder) GetBufferCount() int {
	return h.bufferCount
}

// GetFileSize returns the file size
func (h *HeaderBuilder) GetFileSize() int64 {
	return h.fileSize
}

// GetFilename returns the file name
func (h *HeaderBuilder) GetFilename() string {
	return h.filename
}
