package tcp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/logger"
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
	trjWsPath      string
	erred          bool
	logger         *logger.BaseLogger
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

func (fwd *WebForwarder) SetTrjConfig(trjAddress, trjWsPath string) {
	fwd.trjAddress = trjAddress
	fwd.trjWsPath = trjWsPath
}

func (fwd *WebForwarder) SetSNI(sni string) {
	fwd.sni = sni
}

func (fwd *WebForwarder) SetLogger(l *logger.BaseLogger) {
	fwd.logger = l
}

func (fwd *WebForwarder) Start() {
	if fwd.logger.LogLevel >= logger.Debug {
		defer CloseConnectionDebug(fwd.srcConn)
	} else {
		defer CloseConnection(fwd.srcConn)
	}

	fwd.logger.PrintInfo(fmt.Sprintf("%s opened from %s\n", fwd.connInfoPrefix, fwd.srcConn.RemoteAddr()))

	buff := make([]byte, fwd.bufferSize)
	nr, err := fwd.srcConn.Read(buff)
	b := buff[0:nr]

	var reqArr []string
	isWs := false
	buffScanner := bufio.NewScanner(strings.NewReader(string(b)))
	for buffScanner.Scan() {
		if strings.ToLower(buffScanner.Text()) == "upgrade: websocket" {
			isWs = true
		}
		reqArr = append(reqArr, buffScanner.Text())
	}

	if !isWs {
		fwd.srcConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\nConnection: close\r\n\r\nNo valid websocket request"))
		fwd.logger.PrintInfo(fmt.Sprintf("%s closed\n", fwd.connInfoPrefix))
		return
	}

	remoteKind := "ssh"
	remoteAddress := fwd.dstAddress
	if strings.Contains(reqArr[0], fmt.Sprintf("%s ", fwd.trjWsPath)) {
		remoteAddress = fwd.trjAddress
		remoteKind = "trojan"
		// rewrite request path to trojan path
		reqArr[0] = strings.Replace(reqArr[0], strings.Split(reqArr[0], " ")[1], fwd.trjWsPath, -1)
	}

	fwd.logger.PrintInfo(fmt.Sprintf("%s websocket (%s) session opened from %s\n", fwd.connInfoPrefix, remoteKind, fwd.srcConn.RemoteAddr()))

	if fwd.secure || (!fwd.secure && remoteKind != "ssh") {
		fwd.dstConn, err = tls.Dial("tcp", remoteAddress, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         fwd.sni,
		})
	} else {
		fwd.dstConn, err = net.Dial("tcp", remoteAddress)
	}
	if err != nil {
		fwd.logger.PrintCritical(fmt.Sprintf("%s cannot connect to backend '%s'\n", fwd.connInfoPrefix, err))
		return
	}
	if fwd.logger.LogLevel >= logger.Debug {
		defer CloseConnectionDebug(fwd.dstConn)
	} else {
		defer CloseConnection(fwd.dstConn)
	}

	// initial forward tcp connection to backend
	fwd.dstConn.Write(b)
	fwd.logger.PrintDebug(fmt.Sprintf("%s request\n", fwd.connInfoPrefix))
	fwd.logger.PrintDebug(fmt.Sprintf("%s\n", string(b)))

	go fwd.handleForwardData(fwd.dstConn, fwd.srcConn)
	go fwd.handleForwardData(fwd.srcConn, fwd.dstConn)
	<-fwd.errCh

	fwd.logger.PrintInfo(fmt.Sprintf("%s closed\n", fwd.connInfoPrefix))
}

func (fwd *WebForwarder) handleForwardData(src net.Conn, dst net.Conn) {
	buff := make([]byte, fwd.bufferSize)
	for {
		nr, err := src.Read(buff)
		if err != nil {
			fwd.logger.PrintError(fmt.Sprintf("Cannot read buffer '%s'\n", err))
			fwd.err()
			return
		}
		b := buff[0:nr]
		nr, err = dst.Write(b)
		if err != nil {
			fwd.logger.PrintError(fmt.Sprintf("Cannot write buffer '%s'\n", err))
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
