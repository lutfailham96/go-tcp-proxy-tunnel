package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/tcp"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/util"
	"net"
	"os"
	"sync"
)

var (
	httpAddress    = flag.String("l", "0.0.0.0:80", "http listen address")
	httpsAddress   = flag.String("ln", "0.0.0.0:443", "https listen address")
	tlsCert        = flag.String("cert", "", "tls cert pem")
	tlsKey         = flag.String("key", "", "tls key pem")
	backendAddress = flag.String("b", "127.0.0.1:8082", "backend proxy address")
	trojanAddress  = flag.String("t", "127.0.0.1:433", "trojan backend address")
	trojanWsPath   = flag.String("tp", "/ws-trojan", "trojan websocket path")
	sni            = flag.String("sni", "", "server name identification")
)

func main() {
	flag.Parse()

	if *sni == "" {
		fmt.Println("SNI required!")
		os.Exit(1)
	}

	var tcpWg sync.WaitGroup

	tcpWg.Add(2)
	fmt.Printf("SNI:\t\t\t%s\n", *sni)
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
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cer},
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
		fwd.SetTrjConfig(*trojanAddress, *trojanWsPath)
		fwd.SetSNI(*sni)
		go fwd.Start()
	}
}
