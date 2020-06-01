package socket

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

// socketBuffer 리시브용 소켓 버퍼
type socketBuffer struct {
	data 	[]byte
	offset  int
}

// TCP is
type TCP struct {
	connection net.Conn
	connected  bool
	buffer 	   socketBuffer
}

// Listener is for server
type Listener struct {
	ln       net.Listener
	flagStop bool
}

func (b *socketBuffer) initSocketBuffer() {
	b.data = make([]byte, 65536)
}

func (b *socketBuffer) write(p []byte) {
	l := len(p)
	if n := copy(b.data[b.offset:], p); n < l {
		b.data = append(b.data, p[n:]...)
	}

	b.offset = b.offset + len(p)
}

func (b *socketBuffer) peek(size int) ([]byte, error) {
	if size > b.offset {
		return nil, errors.New("Overflow")
	}

	return b.data[:size], nil
}

func (b *socketBuffer) read(buffer []byte, size int) error {
	if size > b.offset {
		return errors.New("Overflow")
	}

	if len(buffer) < size {
		panic("What the fuck..")
	}

	b.offset = b.offset - size
	copy(buffer, b.data[:size])
	copy(b.data, b.data[size:])

	return nil
}

// Connect is
func (t *TCP) Connect(address string, port uint) bool {
	var err error
	host := address + ":" + fmt.Sprint(port)
	t.connection, err = net.Dial("tcp", host)

	if err != nil {
		return false
	}

	t.connected = true
	t.buffer.initSocketBuffer()
	return t.connected
}

// IsConnected is connected or not
func (t *TCP) IsConnected() bool {
	return t.connected
}

// Close 바로 끊김 ㅋ
func (t *TCP) Close() {
	t.connection.Close()
	t.connected = false
}

// DelayClose 곧 끊김 ㅋ
func (t *TCP) DelayClose() {
	closeTimer := time.NewTimer(time.Second * 2)
	go func() {
		<-closeTimer.C
		closeTimer.Stop()
		t.Close()
	}()
}

// GetRemoteAddr 접속중인 peer 의 원격지 주소
func (t *TCP) GetRemoteAddr() string {
	return t.connection.RemoteAddr().String()
}

// ConnectionHandler is
func (t *TCP) ConnectionHandler(f func(), d func()) {
	bufBytes := make([]byte, 65536)
	for {
		n, err := t.connection.Read(bufBytes)
		if err != nil {
			if n == 0 {
				t.connected = false
				log.Println(err)
				d()
			}
			break
		}

		if n > 0 {
			t.buffer.write(bufBytes[:n])
			f()
		}
	}
}

// Send is
func (t *TCP) Send(buf []byte) {
	t.connection.Write(buf)
}

func (t *TCP) Peek(size int) ([]byte, error) {
	return t.buffer.peek(size)
}

func (t *TCP) Read(buffer []byte, size int) error {
	return t.buffer.read(buffer, size)
}

// Listen is for server
func (l *Listener) Listen(port uint) error {
	str := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", str)
	if err != nil {
		return err
	}

	l.ln = ln
	l.flagStop = false

	return nil
}

// AsyncAccept is accept on background
func (l *Listener) AsyncAccept(acceptCallback func(*TCP)) {
	go func() {
		for {
			conn, _ := l.ln.Accept()
			if l.flagStop {
				break
			}
			connection := new(TCP)
			connection.connection = conn
			connection.connected = true
			connection.buffer.initSocketBuffer()

			acceptCallback(connection)
		}
	}()
}

// StopAccept will be stopped service
func (l *Listener) StopAccept() {
	l.flagStop = true
	l.ln.Close()
}
