package tcp

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

type WebForwarder struct {
	secure         bool
	sni            string
	connInfoPrefix string
	connectionId   uint64
	bufferSize     uint64
	errCh          chan bool
	srcConn        net.Conn
	dstConn        net.Conn
	dstAddress     string
	trjAddress     string
	erred          bool
}

func NewWebForwarder(connId uint64, src net.Conn, secure bool) *WebForwarder {
	connInfoPrefix := fmt.Sprintf("CONN #%d", connId)
	if secure {
		connInfoPrefix = fmt.Sprintf("CONN (TLS) #%d", connId)
	}
	return &WebForwarder{
		connectionId:   connId,
		connInfoPrefix: connInfoPrefix,
		bufferSize:     0xffff,
		secure:         false,
		srcConn:        src,
		errCh:          make(chan bool, 2),
		erred:          false,
	}
}

func (fwd *WebForwarder) SetDstAddress(dstAddress string) {
	fwd.dstAddress = dstAddress
}

func (fwd *WebForwarder) SetTrjAddress(trjAddress string) {
	fwd.trjAddress = trjAddress
}

func (fwd *WebForwarder) SetSNI(sni string) {
	fwd.sni = sni
}

func (fwd *WebForwarder) Start() {
	defer CloseConnection(fwd.srcConn)

	fmt.Printf("%s opened from %s\n", fwd.connInfoPrefix, fwd.srcConn.RemoteAddr())

	buff := make([]byte, fwd.bufferSize)
	nr, err := fwd.srcConn.Read(buff)
	b := buff[0:nr]
	if !strings.Contains(strings.ToLower(string(b)), "upgrade: websocket") {
		fwd.srcConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\nConnection: close\r\n\r\nNo valid websocket request"))
		fmt.Printf("%s closed\n", fwd.connInfoPrefix)
		return
	}

	remoteKind := "ssh"
	remoteAddress := fwd.dstAddress
	if strings.Contains(strings.ToLower(string(b)), "/ws-trojan") {
		remoteAddress = fwd.trjAddress
		remoteKind = "trojan"
	}

	fmt.Printf("%s websocket (%s) session opened from %s\n", fwd.connInfoPrefix, remoteKind, fwd.srcConn.RemoteAddr())

	if fwd.secure && remoteKind != "ssh" {
		fwd.dstConn, err = tls.Dial("tcp", remoteAddress, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         fwd.sni,
		})
	} else {
		fwd.dstConn, err = net.Dial("tcp", remoteAddress)
	}
	if err != nil {
		fmt.Printf("%s cannot connect to backend '%s'\n", fwd.connInfoPrefix, err)
		return
	}
	defer CloseConnection(fwd.dstConn)

	// initial forward tcp connection to backend
	fwd.dstConn.Write(b)
	fmt.Printf("%s request\n", fwd.connInfoPrefix)
	fmt.Println(string(b))

	go fwd.handleForwardData(fwd.dstConn, fwd.srcConn)
	go fwd.handleForwardData(fwd.srcConn, fwd.dstConn)
	<-fwd.errCh

	fmt.Printf("%s closed\n", fwd.connInfoPrefix)
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
	if fwd.erred {
		return
	}
	fwd.errCh <- true
	fwd.erred = true
}
