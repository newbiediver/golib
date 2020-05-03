package socket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

// TCP is
type TCP struct {
	connection net.Conn
	connected  bool
}

// Listener is for server
type Listener struct {
	ln       net.Listener
	flagStop bool
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
	return t.connected
}

// IsConnected is connected or not
func (t *TCP) IsConnected() bool {
	return t.connected
}

// Close is
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

// ConnectionHandler is
func (t *TCP) ConnectionHandler(f func([]byte), d func()) {
	bufBytes := make([]byte, 32768)
	for {
		buf := bufio.NewReader(t.connection)
		n, err := buf.Read(bufBytes)
		if err != nil {
			if n == 0 {
				t.connected = false
				log.Println(err)
				d()
			}
			break
		}

		if n > 0 {
			f(bufBytes[:n])
		}
	}
}

// Send is
func (t *TCP) Send(buf []byte) {
	t.connection.Write(buf)
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

			acceptCallback(connection)
		}
	}()
}

// StopAccept will be stopped service
func (l *Listener) StopAccept() {
	l.flagStop = true
	l.ln.Close()
}
