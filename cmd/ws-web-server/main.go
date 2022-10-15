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
	backendAddress = flag.String("b", "127.0.0.1:8082", "backend proxy address")
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
		tlsConfig, _, err := util.TLSGenerateConfig()
		if err != nil {
			fmt.Printf("Cannot setup tls certificates '%s'", err)
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
		fwd := tcp.NewWebForwarder(connId, src)
		fwd.SetDstAddress(*backendAddress)
		go fwd.Start()
	}
}
