package pkg

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// UDPWaiter represents a UDP component for receiving data
type UDPWaiter struct {
	port   int
	listen string

	receivedData chan ReceivedData
	eof          chan interface{}
}

// ReceivedData is the data read from UDP
type ReceivedData struct {
	Data   []byte
	Remote *net.UDPAddr
}

// NewUDPWaiter creates an instance of NewUDPWaiter
func NewUDPWaiter(port int) *UDPWaiter {
	return &UDPWaiter{
		port:         port,
		listen:       "0.0.0.0",
		receivedData: make(chan ReceivedData, 1024),
		eof:          make(chan interface{}),
	}
}

// ListenAddress set the listen address
func (w *UDPWaiter) ListenAddress(address string) *UDPWaiter {
	w.listen = address
	return w
}

// Start starts UDP connection
func (w *UDPWaiter) Start(msg chan string) (err error) {
	udpAddress := &net.UDPAddr{
		Port: w.port,
		IP:   net.ParseIP(w.listen),
	}
	defer close(msg)

	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	msg <- fmt.Sprintf("server listening %s\n", conn.LocalAddr().String())
	header, err := readHeader(conn)
	if err != nil {
		return err
	}
	msg <- fmt.Sprintf("start to receive data from %v\n", header.remote)

	f, err := os.OpenFile(header.filename, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err = f.Write(make([]byte, header.length)); err != nil {
		err = fmt.Errorf("failed to init file, %v", err)
		return err
	}

	mapBuffer := NewSafeMap(header.count)
	go func() {
		if _, err := f.WriteAt(header.data, int64(header.chrunk*header.index)); err == nil {
			mapBuffer.Remove(header.index)
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		//startedMissingThread := false
		for size := mapBuffer.Size(); size > 0; size = mapBuffer.Size() {
			message := make([]byte, 65507)
			data := ReceivedData{}
			var (
				rlen    int
				readErr error
			)
			if rlen, data.Remote, readErr = conn.ReadFromUDP(message[:]); readErr == nil {
				data.Data = message[:rlen]
				w.receivedData <- data
			}
		}
		// notify all the data was received
		w.eof <- nil
	}()

	// start a thread to write the data
	go func() {
		for {
			select {
			case <-w.eof:
				return
			case data := <-w.receivedData:
				writeData(data, f, mapBuffer)
			}
		}
	}()

	// check the buffer and start the missing thread
	lastCount := 0
	time.Sleep(time.Microsecond * time.Duration(lastCount) * 50)
	for lastCount != mapBuffer.Size() {
		lastCount = mapBuffer.Size()
		time.Sleep(time.Second * 5)
	}
	sendWaitingMissingRequest(&wg, &header, mapBuffer, conn, msg)

	wg.Wait()
	msg <- fmt.Sprintf("wrote to file %s\n", f.Name())
	return
}

func writeData(data ReceivedData, f *os.File, mapBuffer *SafeMap) {
	header, err := readHeaderFromData(data)
	if err == nil {
		_, err = f.WriteAt(header.data, int64(header.chrunk*header.index))
		if err == nil {
			mapBuffer.Remove(header.index)
		}
	}
}

func sendWaitingMissingRequest(wg *sync.WaitGroup, header *dataHeader, buffer *SafeMap, conn *net.UDPConn, msg chan string) {
	wg.Add(1)
	go func() {
		defer func() {
			_ = conn.Close()
			wg.Done()
		}()

		for buffer.Size() > 0 {
			missing := buffer.GetKeys()
			//fmt.Println("missing", len(missing))
			for _, i := range missing {
				_ = requestMissing(conn, i, header.remote)
			}
			time.Sleep(time.Second)
		}

		for err := requestDone(conn, header.remote); err != nil; {
		}
		msg <- "done with checking\n"
	}()
}

func requestDone(conn *net.UDPConn, remote *net.UDPAddr) (err error) {
	for i := 0; i < 3; i++ {
		_, err = conn.WriteTo([]byte("done"+fillContainerWithNumber(0, 10)), remote)

		time.Sleep(time.Second)
	}
	return
}

func requestMissing(conn *net.UDPConn, index int, remote *net.UDPAddr) (err error) {
	_, err = conn.WriteTo([]byte("miss"+fillContainerWithNumber(index, 10)), remote)
	return
}
