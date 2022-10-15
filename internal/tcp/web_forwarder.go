package tcp

import (
	"fmt"
	"net"
	"strings"
)

type WebForwarder struct {
	connectionId uint64
	bufferSize   uint64
	errCh        chan bool
	srcConn      net.Conn
	dstConn      net.Conn
	dstAddress   string
}

func NewWebForwarder(connId uint64, src net.Conn) *WebForwarder {
	return &WebForwarder{
		connectionId: connId,
		bufferSize:   0xffff,
		srcConn:      src,
		errCh:        make(chan bool, 1),
	}
}

func (fwd *WebForwarder) SetDstAddress(dstAddress string) {
	fwd.dstAddress = dstAddress
}

func (fwd *WebForwarder) Start() {
	defer CloseConnection(fwd.srcConn)

	fmt.Printf("CONN #%d opened from %s\n", fwd.connectionId, fwd.srcConn.RemoteAddr())

	buff := make([]byte, fwd.bufferSize)
	nr, err := fwd.srcConn.Read(buff)
	b := buff[0:nr]
	if !strings.Contains(strings.ToLower(string(b)), "upgrade: websocket") {
		fwd.srcConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\nConnection: close\r\n\r\nNo valid websocket request"))
		return
	}

	fmt.Printf("CONN #%d websocket session opened from %s\n", fwd.connectionId, fwd.srcConn.RemoteAddr())

	fwd.dstConn, err = net.Dial("tcp", fwd.dstAddress)
	if err != nil {
		fmt.Printf("Cannot connect to backend '%s'", err)
		return
	}
	defer CloseConnection(fwd.dstConn)

	// initial forward tcp connection to backend
	fwd.dstConn.Write(b)

	go fwd.handleForwardData(fwd.dstConn, fwd.srcConn)
	go fwd.handleForwardData(fwd.srcConn, fwd.dstConn)
	<-fwd.errCh

	fmt.Printf("CONN #%d closed\n", fwd.connectionId)
}

func (fwd *WebForwarder) handleForwardData(src net.Conn, dst net.Conn) {
	buff := make([]byte, fwd.bufferSize)
	for {
		nr, err := src.Read(buff)
		if err != nil {
			//fmt.Printf("Cannot read buffer '%s'\n", err)
			fwd.err()
			return
		}
		b := buff[0:nr]
		nr, err = dst.Write(b)
		if err != nil {
			//fmt.Printf("Cannot write buffer '%s'\n", err)
			fwd.err()
			return
		}
	}
}

func (fwd *WebForwarder) err() {
	fwd.errCh <- true
}
