package main

import (
	"flag"
	"fmt"
	proxy "github.com/lutfailham96/go-tcp-proxy-tunnel"
	"net"
)

var (
	localAddr        = flag.String("l", "127.0.0.1:8082", "local address")
	remoteAddr       = flag.String("r", "127.0.0.1:443", "remote address")
	serverHost       = flag.String("s", "", "server host address")
	reverseProxyMode = flag.Bool("rp", false, "enable reverse proxy mode")
	localPayload     = flag.String("ip", "", "local TCP payload replacer")
	remotePayload    = flag.String("op", "", "remote TCP payload replacer")
)

func main() {
	flag.Parse()

	lAddr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		fmt.Printf("Failed to resolve local address: %s", err)
		return
	}

	rAddr, err := net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		fmt.Printf("Failed to resolve remote address: %s", err)
		return
	}

	if *serverHost != "" {
		_, err = net.ResolveTCPAddr("tcp", *serverHost)
		if err != nil {
			fmt.Printf("Failed to resolve remote address: %s", err)
			return
		}
	}

	listener, err := net.Listen("tcp", lAddr.String())
	if err != nil {
		fmt.Printf("Failed to open local port to listen: %s", err)
		return
	}
	fmt.Printf("go-tcp-proxy proxing from %v to %v\n", lAddr, rAddr)

	var connId = uint64(0)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection '%s'", err)
			return
		}
		connId += 1

		var p *proxy.Proxy
		p = p.New(connId, conn, lAddr, rAddr)
		if *serverHost != "" {
			p.SetServerHost(*serverHost)
		}
		p.SetlPayload(*localPayload)
		p.SetrPayload(*remotePayload)
		p.SetReverseProxy(*reverseProxyMode)
		go p.Start()
	}
}
