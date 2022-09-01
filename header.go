package main

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

func readHeader(conn *net.UDPConn) (header dataHeader, err error) {
	message := make([]byte, 65507)
	var rlen int
	rlen, header.remote, err = conn.ReadFromUDP(message[:])
	if err == nil {
		if rlen <= 150 {
			err = fmt.Errorf("invalid header format")
			return
		}

		if header.length, err = strconv.Atoi(strings.TrimSpace(string(message[:20]))); err != nil {
			return
		}
		header.filename = strings.TrimSpace(string(message[20:120]))
		if header.chrunk, err = strconv.Atoi(strings.TrimSpace(string(message[120:130]))); err != nil {
			return
		}
		if header.count, err = strconv.Atoi(strings.TrimSpace(string(message[130:140]))); err != nil {
			return
		}
		if header.index, err = strconv.Atoi(strings.TrimSpace(string(message[140:150]))); err != nil {
			return
		}

		header.data = message[150:rlen]
	}
	return
}

type headerBuilder struct {
	file string

	filename    string
	fileSize    int64
	chunk       int
	bufferCount int
}

// NewHeaderBuilder creates an instance of the headerBuilder
func NewHeaderBuilder(file string) *headerBuilder {
	return &headerBuilder{
		file: file,
	}
}

func (h *headerBuilder) Build() (err error) {
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
func (h *headerBuilder) CreateHeader(index int, data []byte) []byte {
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
func (h *headerBuilder) GetChunk() int {
	return h.chunk
}

// GetBufferCount returns the buffer count
func (h *headerBuilder) GetBufferCount() int {
	return h.bufferCount
}

// GetFileSize returns the file size
func (h *headerBuilder) GetFileSize() int64 {
	return h.fileSize
}

// GetFilename returns the file name
func (h *headerBuilder) GetFilename() string {
	return h.filename
}
