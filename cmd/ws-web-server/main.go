package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/tcp"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/util"
	"net"
	"sync"
)

var (
	httpAddress    = flag.String("l", "0.0.0.0:80", "http listen address")
	httpsAddress   = flag.String("ln", "0.0.0.0:443", "https listen address")
	tlsCert        = flag.String("cert", "", "tls cert pem")
	tlsKey         = flag.String("key", "", "tls key pem")
	backendAddress = flag.String("b", "127.0.0.1:8082", "backend proxy address")
	trojanAddress  = flag.String("t", "127.0.0.1:433", "trojan backend address")
)

func main() {
	flag.Parse()

	var tcpWg sync.WaitGroup

	tcpWg.Add(2)
	go setupTcpListener(false)
	go setupTcpListener(true)

	tcpWg.Wait()
}

func setupTcpListener(secure bool) {
	var ln net.Listener
	var err error

	if secure {
		var tlsConfig *tls.Config
		if *tlsCert != "" && *tlsKey != "" {
			cer, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
			if err != nil {
				fmt.Printf("Cannot read tls key pair '%s'\n", err)
			}
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cer},
			}
		} else {
			tlsConfig, _, err = util.TLSGenerateConfig()
			if err != nil {
				fmt.Printf("Cannot setup tls certificates '%s'\n", err)
			}
		}
		tcp.ResolveAddr(*httpsAddress)
		ln, err = tls.Listen("tcp", *httpsAddress, tlsConfig)
		fmt.Printf("Secure TCP listen on:\t%s\n", *httpsAddress)
	} else {
		tcp.ResolveAddr(*httpAddress)
		ln, err = net.Listen("tcp", *httpAddress)
		fmt.Printf("TCP listen on:\t\t%s\n", *httpAddress)
	}
	if err != nil {
		fmt.Printf("Cannot bind port '%s'\n", err)
		return
	}

	connId := uint64(0)
	for {
		src, err := ln.Accept()
		if err != nil {
			fmt.Printf("Cannot accept connection '%s'\n", err)
			continue
		}
		connId += 1
		fwd := tcp.NewWebForwarder(connId, src, secure)
		fwd.SetDstAddress(*backendAddress)
		fwd.SetTrjAddress(*trojanAddress)
		go fwd.Start()
	}
}
