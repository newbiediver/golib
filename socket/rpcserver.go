package socket

import (
	"encoding/binary"
	"errors"
	"github.com/newbiediver/golib/xlog"
	"log"
	"strings"
	"sync"
	"time"
)

type rpcHeader struct {
	rpcSize    uint64
	bodySize   uint64
	rpcNameLen uint64
}

type RPCClient struct {
	connector *TCP
}

type RPCServer struct {
	lock            *sync.Mutex
	listener        *Listener
	clientContainer map[*TCP]*RPCClient
	eventFunctor    func(*RPCClient, string, []string)
	xlogUsing       bool
}

const (
	logInfo int = 0 + iota
	logWarn
	logError
	logFatal
)

func extractRpc(connector *TCP, buffer []byte) []byte {
	const headerSize int = 8
	rawHeader, err := connector.Peek(headerSize)
	if err != nil {
		return nil
	}

	size := binary.LittleEndian.Uint64(rawHeader)
	err = connector.Read(buffer, int(size))
	if err != nil {
		return nil
	}

	return buffer
}

func (r *RPCServer) RunServer(port uint16) error {
	r.lock = new(sync.Mutex)
	r.listener = new(Listener)
	r.clientContainer = make(map[*TCP]*RPCClient)

	if err := r.listener.Listen(uint(port)); err != nil {
		return errors.New("Initializing RPC port is failed")
	}

	r.listener.AsyncAccept(func(connector *TCP) {
		rpc := new(RPCClient)
		rpc.connector = connector
		r.addClient(rpc)
		r.rpcLog(logInfo, "Connected rpc client: %s", connector.GetRemoteAddr())

		go func() {
			rpcBuffer := make([]byte, 32768)
			connector.ConnectionHandler(func() {
				for extractRpc(connector, rpcBuffer) != nil {
					r.rpcReceiver(connector, rpcBuffer)
				}
			}, func() {
				r.deleteClient(connector)
				r.rpcLog(logWarn, "Disconnected rpc client: %s", connector.GetRemoteAddr())
			})
		}()
	})

	return nil
}

func (r *RPCServer) StopServer() {
	r.lock.Lock()
	for connector, _ := range r.clientContainer {
		connector.Close()
	}
	r.lock.Unlock()

	for len(r.clientContainer) > 0 {
		time.Sleep(time.Millisecond)
	}

	r.listener.StopAccept()
}

func (r *RPCServer) UseXlog() {
	r.xlogUsing = true
}

func (r *RPCServer) SetEventFunctor(functor func(*RPCClient, string, []string)) {
	r.eventFunctor = functor
}

func (r *RPCClient) Send(str string) {
	const objSize int = 24

	obj := rpcHeader{
		rpcSize:    uint64(len(str) + objSize),
		bodySize:   uint64(len(str)),
		rpcNameLen: 0,
	}

	p := make([]byte, objSize)
	binary.LittleEndian.PutUint64(p[0:8], obj.rpcSize)
	binary.LittleEndian.PutUint64(p[8:16], obj.bodySize)
	binary.LittleEndian.PutUint64(p[16:24], 0)

	p = append(p, []byte(str)...)

	r.connector.Send(p)
}

func (r *RPCServer) addClient(c *RPCClient) {
	defer r.lock.Unlock()

	r.lock.Lock()
	r.clientContainer[c.connector] = c
}

func (r *RPCServer) deleteClient(connector *TCP) {
	defer r.lock.Unlock()

	r.lock.Lock()
	delete(r.clientContainer, connector)
}

func (r *RPCServer) rpcLog(lv int, format string, a ...interface{}) {
	if r.xlogUsing {
		switch lv {
		case logInfo:
			xlog.Info(format, a)
		case logWarn:
			xlog.Warn(format, a)
		case logError:
			xlog.Error(format, a)
		case logFatal:
			xlog.Fatal(format, a)
		}
	} else {
		log.Printf(format, a, "\n")
	}
}

func (r *RPCServer) rpcReceiver(connector *TCP, p []byte) {
	rpcSession := r.clientContainer[connector]
	if rpcSession == nil {
		r.rpcLog(logError, "Invalid rpc session")
		return
	}

	rawBodySize := p[8:16]
	rawFuncNameSize := p[16:24]
	funcNameLen := binary.LittleEndian.Uint64(rawFuncNameSize)
	bodySize := binary.LittleEndian.Uint64(rawBodySize)

	funcName := string(p[24 : 24+funcNameLen])
	body := p[24+funcNameLen : 24+funcNameLen+bodySize]
	args := r.parseArgs(body)

	if r.eventFunctor != nil {
		r.eventFunctor(rpcSession, funcName, args)
	}
}

func (r *RPCServer) parseArgs(bytes []byte) []string {
	defer func() {
		if rc := recover(); rc != nil {
			r.rpcLog(logFatal, "%s", r)
		}
	}()

	var args []string
	bodyString := string(bytes)
	if bodyString[0] != '"' && bodyString[len(bodyString)-1] != '"' {
		args = strings.Split(bodyString, ",")
	} else {
		args = append(args, bodyString[1:len(bodyString)-1])
	}

	return args
}
