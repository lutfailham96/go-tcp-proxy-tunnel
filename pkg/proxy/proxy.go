package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/logger"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/tcp"
)

type Proxy struct {
	secure               bool
	connectionInfoPrefix string
	proxyKind            string
	conn                 net.Conn
	lConn                net.Conn
	rConn                net.Conn
	lAddr                *net.TCPAddr
	rAddr                *net.TCPAddr
	sHost                tcp.Host
	tlsEnabled           bool
	sniHost              string
	lPayload             []byte
	rPayload             []byte
	buffSize             uint64
	lInitialized         bool
	rInitialized         bool
	bytesReceived        uint64
	bytesSent            uint64
	erred                bool
	errSig               chan bool
	connId               uint64
	serverProxyMode      bool
	wsUpgradeInitialized bool
	logger               *logger.BaseLogger
}

func NewProxy(connId uint64, conn net.Conn, lAddr, rAddr *net.TCPAddr, secure bool) *Proxy {
	return &Proxy{
		secure:               secure,
		conn:                 conn,
		lConn:                conn,
		lAddr:                lAddr,
		rAddr:                rAddr,
		lPayload:             make([]byte, 0),
		rPayload:             make([]byte, 0),
		buffSize:             uint64(0xffff),
		lInitialized:         false,
		rInitialized:         false,
		erred:                false,
		errSig:               make(chan bool, 2),
		connId:               connId,
		serverProxyMode:      false,
		wsUpgradeInitialized: false,
	}
}

func (p *Proxy) SetlPayload(lPayload string) {
	if p.sHost.HostName != "" {
		lPayload = strings.Replace(lPayload, "[host]", p.sHost.HostName, -1)
		lPayload = strings.Replace(lPayload, "[host_port]", fmt.Sprintf("%s:%d", p.sHost.HostName, p.sHost.Port), -1)
	}
	if p.sniHost != "" {
		lPayload = strings.Replace(lPayload, "[sni]", p.sniHost, -1)
	}
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

func (p *Proxy) SetServerProxyMode(enabled bool) {
	p.serverProxyMode = enabled
}

func (p *Proxy) SetServerHost(server string) {
	sHost, sPort, err := net.SplitHostPort(server)
	if err != nil {
		p.logger.PrintCritical(fmt.Sprintf("%s cannot parse server host port '%s'\n", p.connectionInfoPrefix, err))
		return
	}
	sPortParsed, err := strconv.ParseUint(sPort, 10, 64)
	if err != nil {
		p.logger.PrintCritical(fmt.Sprintf("%s cannot parse server port '%s'\n", p.connectionInfoPrefix, err))
		return
	}
	p.sHost = tcp.Host{
		HostName: sHost,
		Port:     sPortParsed,
	}
}

func (p *Proxy) SetBufferSize(buffSize uint64) {
	p.buffSize = buffSize
}

func (p *Proxy) SetEnableTLS(enabled bool) {
	p.tlsEnabled = enabled
}

func (p *Proxy) SetSNIHost(hostname string) {
	p.sniHost = hostname
}

func (p *Proxy) SetProxyKind(proxyKind string) {
	p.proxyKind = proxyKind
	connInfoPrefix := fmt.Sprintf("CONN %s #%d", p.proxyKind, p.connId)
	if p.secure {
		connInfoPrefix = fmt.Sprintf("CONN %s (TLS) #%d", p.proxyKind, p.connId)
	}
	p.connectionInfoPrefix = connInfoPrefix
}

func (p *Proxy) SetLogger(logger *logger.BaseLogger) {
	p.logger = logger
}

func (p *Proxy) Start() {
	if p.logger.LogLevel >= logger.Debug {
		defer func(conn net.Conn) {
			tcp.CloseConnectionDebug(conn)
		}(p.lConn)
	} else {
		defer func(conn net.Conn) {
			tcp.CloseConnection(conn)
		}(p.lConn)
	}

	var err error
	if p.tlsEnabled {
		p.rConn, err = tls.Dial("tcp", p.rAddr.String(), &tls.Config{
			ServerName:         p.sniHost,
			InsecureSkipVerify: true,
		})
	} else {
		p.rConn, err = net.DialTCP("tcp", nil, p.rAddr)
	}
	if err != nil {
		p.logger.PrintCritical(fmt.Sprintf("%s cannot dial remote connection '%s'\n", p.connectionInfoPrefix, err))
		return
	}
	if p.logger.LogLevel >= logger.Debug {
		defer func(conn net.Conn) {
			tcp.CloseConnectionDebug(conn)
		}(p.rConn)
	} else {
		defer func(conn net.Conn) {
			tcp.CloseConnection(conn)
		}(p.rConn)
	}

	p.logger.PrintInfo(fmt.Sprintf("%s opened %s >> %s\n", p.connectionInfoPrefix, p.lAddr, p.rAddr))

	go p.handleForwardData(p.lConn, p.rConn)
	if !p.serverProxyMode {
		go p.handleForwardData(p.rConn, p.lConn)
	}
	<-p.errSig
	p.logger.PrintInfo(fmt.Sprintf("%s closed (%d bytes sent, %d bytes received)\n", p.connectionInfoPrefix, p.bytesSent, p.bytesReceived))
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
	buffer := make([]byte, p.buffSize)

	for {
		n, err := src.Read(buffer)
		if err != nil {
			p.logger.PrintError(fmt.Sprintf("Cannot read buffer from source '%s'\n", err))
			p.err()
			return
		}
		connBuff := buffer[:n]
		if isLocal {
			p.handleOutboundData(src, dst, &connBuff)
		} else {
			p.handleInboundData(src, dst, &connBuff)
		}
		if p.serverProxyMode && p.wsUpgradeInitialized {
			n, err = src.Write(connBuff)
			p.wsUpgradeInitialized = false
			go p.handleForwardData(dst, src)
		} else {
			n, err = dst.Write(connBuff)
		}
		if err != nil {
			p.logger.PrintError(fmt.Sprintf("Cannot write buffer to destination '%s'\n", err))
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

func (p *Proxy) handleOutboundData(src, dst net.Conn, connBuff *[]byte) {
	if p.lInitialized {
		return
	}

	p.logger.PrintInfo(fmt.Sprintf("%s %s >> %s >> %s\n", p.connectionInfoPrefix, src.RemoteAddr(), p.conn.LocalAddr(), dst.RemoteAddr()))

	var respArr []string
	doUpgrade := false
	buffScanner := bufio.NewScanner(strings.NewReader(string(*connBuff)))
	for buffScanner.Scan() {
		respArr = append(respArr, buffScanner.Text())
		if strings.Contains(strings.ToLower(buffScanner.Text()), "upgrade: websocket") {
			doUpgrade = true
		}
	}

	if p.serverProxyMode {
		if doUpgrade {
			p.logger.PrintInfo(fmt.Sprintf("%s connection upgrade to Websocket\n", p.connectionInfoPrefix))
			*connBuff = []byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
			p.wsUpgradeInitialized = true
		}
	} else {
		if p.proxyKind == "ssh" && strings.Contains(respArr[0], "CONNECT ") {
			*connBuff = p.lPayload
			p.logger.PrintDebug(fmt.Sprintf("%s\n", string(*connBuff)))
		}
		if p.proxyKind == "trojan" {
			reqPath := strings.Split(respArr[0], " ")[1]
			newReqPath := fmt.Sprintf(" wss://%s%s ", p.sniHost, reqPath)
			*connBuff = []byte(strings.Replace(string(*connBuff), fmt.Sprintf(" %s ", reqPath), newReqPath, -1))
			p.logger.PrintDebug(fmt.Sprintf("%s\n", string(*connBuff)))
		}
	}

	p.lInitialized = true
}

func (p *Proxy) handleInboundData(src, dst net.Conn, connBuff *[]byte) {
	if p.rInitialized {
		return
	}

	p.logger.PrintInfo(fmt.Sprintf("%s %s << %s << %s\n", p.connectionInfoPrefix, dst.RemoteAddr(), p.conn.LocalAddr(), src.RemoteAddr()))

	var respArr []string
	buffScanner := bufio.NewScanner(strings.NewReader(string(*connBuff)))
	for buffScanner.Scan() {
		respArr = append(respArr, buffScanner.Text())
	}
	if strings.Contains(respArr[0], " 101 ") && p.proxyKind == "ssh" {
		respArr[0] = strings.Replace(string(p.rPayload), "\r\n", "", -1)
	}
	// TODO handle redirect 301 / 302
	//if strings.Contains(respArr[0], " 301 ") || strings.Contains(respArr[0], "302") {
	//	respArr[0] = "HTTP/1.1 101 Switching Protocols"
	//}

	if !p.serverProxyMode {
		*connBuff = []byte(strings.Join(respArr, "\r\n") + "\r\n")
		p.logger.PrintDebug(fmt.Sprintf("%s\n", string(*connBuff)))
	}

	p.rInitialized = true
}
