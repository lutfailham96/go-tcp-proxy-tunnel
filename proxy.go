package proxy

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

type Proxy struct {
	conn                 net.Conn
	lConn                net.Conn
	rConn                net.Conn
	lAddr                *net.TCPAddr
	rAddr                *net.TCPAddr
	lPayload             []byte
	rPayload             []byte
	lInitialized         bool
	rInitialized         bool
	bytesReceived        uint64
	bytesSent            uint64
	erred                bool
	errSig               chan bool
	connId               uint64
	reverseProxy         bool
	wsUpgradeInitialized bool
}

func (p *Proxy) New(connId uint64, conn net.Conn, lAddr, rAddr *net.TCPAddr) *Proxy {
	return &Proxy{
		conn:                 conn,
		lConn:                conn,
		lAddr:                lAddr,
		rAddr:                rAddr,
		lPayload:             make([]byte, 0),
		rPayload:             make([]byte, 0),
		lInitialized:         false,
		rInitialized:         false,
		erred:                false,
		errSig:               make(chan bool),
		connId:               connId,
		reverseProxy:         false,
		wsUpgradeInitialized: false,
	}
}

func (p *Proxy) SetlPayload(lPayload string) {
	lPayload = strings.Replace(lPayload, "[crlf]", "\r\n", -1)
	p.lPayload = []byte(lPayload)
}

func (p *Proxy) SetrPayload(rPayload string) {
	if rPayload == "" {
		rPayload = "HTTP/1.1 200 Connection Established[crlf][crlf]"
	}
	rPayload = strings.Replace(rPayload, "[crlf]", "\r\n", -1)
	p.rPayload = []byte(rPayload)
}

func (p *Proxy) SetReverseProxy(enabled bool) {
	p.reverseProxy = enabled
}

func (p *Proxy) Start() {
	defer p.closeConnection(p.lConn)

	var err error
	p.rConn, err = net.DialTCP("tcp", nil, p.rAddr)
	if err != nil {
		fmt.Printf("Cannot dial remote connection '%s'", err)
		return
	}
	defer p.closeConnection(p.rConn)

	fmt.Printf("CONN #%d opened %s >> %s\n", p.connId, p.lAddr, p.rAddr)

	go p.handleForwardData(p.lConn, p.rConn)
	if !p.reverseProxy {
		go p.handleForwardData(p.rConn, p.lConn)
	}
	<-p.errSig
	fmt.Printf("CONN #%d closed (%d bytes sent, %d bytes received)\n", p.connId, p.bytesSent, p.bytesReceived)
}

func (p *Proxy) err() {
	if p.erred {
		return
	}
	p.errSig <- true
	p.erred = true
}

func (p *Proxy) handleForwardData(src, dst net.Conn) {
	isLocal := src == p.lConn
	buffer := make([]byte, 0xffff)

	for {
		n, err := src.Read(buffer)
		if err != nil {
			//fmt.Printf("Cannot read buffer from source '%s'", err)
			p.err()
			return
		}
		b := buffer[:n]
		if isLocal {
			if !p.lInitialized {
				fmt.Printf("CONN #%d %s >> %s >> %s\n", p.connId, src.RemoteAddr(), p.conn.LocalAddr(), dst.RemoteAddr())
				if p.reverseProxy {
					if strings.Contains(strings.ToLower(string(b)), "upgrade: websocket") {
						fmt.Printf("CONN #%d connection upgrade to Websocket\n", p.connId)
						b = []byte("HTTP/1.1 101 Switching Protocols\r\n\r\n")
						p.wsUpgradeInitialized = true
					}
				} else {
					if bytes.Contains(b, []byte("CONNECT ")) {
						b = p.lPayload
						fmt.Println(string(b))
					}
				}
				p.lInitialized = true
			}
		} else {
			if !p.rInitialized {
				fmt.Printf("CONN #%d %s << %s << %s\n", p.connId, dst.RemoteAddr(), p.conn.LocalAddr(), src.RemoteAddr())
				if bytes.Contains(b, []byte("HTTP/1.")) && !p.reverseProxy {
					b = p.rPayload
					fmt.Println(string(b))
				}
				p.rInitialized = true
			}
		}
		if p.reverseProxy && p.wsUpgradeInitialized {
			n, err = src.Write(b)
			p.wsUpgradeInitialized = false
			go p.handleForwardData(dst, src)
		} else {
			n, err = dst.Write(b)
		}
		if err != nil {
			//fmt.Printf("Cannot write buffer to destination '%s'", err)
			p.err()
			return
		}

		if isLocal {
			p.bytesSent += uint64(n)
		} else {
			p.bytesReceived += uint64(n)
		}
	}
}

func (p *Proxy) closeConnection(conn net.Conn) {
	err := conn.Close()
	if err != nil {
		fmt.Printf("Cannot close connection '%s'", err)
		return
	}
}
