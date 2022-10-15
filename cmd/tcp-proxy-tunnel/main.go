package main

import (
	"flag"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/common"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/pkg/proxy"
	"net"
)

var (
	localAddr           = flag.String("l", "127.0.0.1:8082", "local address")
	remoteAddr          = flag.String("r", "127.0.0.1:443", "remote address")
	serverHost          = flag.String("s", "", "server host address")
	disableServerResolv = flag.Bool("dsr", false, "disable server host resolve")
	serverProxyMode     = flag.Bool("sv", false, "run on server mode")
	localPayload        = flag.String("op", "", "local TCP payload replacer")
	remotePayload       = flag.String("ip", "", "remote TCP payload replacer")
	bufferSize          = flag.Uint64("bs", 0, "connection buffer size")
	tlsEnabled          = flag.Bool("tls", false, "enable tls/secure connection")
	sniHost             = flag.String("sni", "", "SNI hostname")
	configFile          = flag.String("c", "", "load config from JSON file")
)

func main() {
	flag.Parse()

	cmdArgs := &common.CmdArgs{
		LocalAddress:        *localAddr,
		RemoteAddress:       *remoteAddr,
		ServerHost:          *serverHost,
		DisableServerResolv: *disableServerResolv,
	}

	config := &common.Config{
		ServerProxyMode:     *serverProxyMode,
		BufferSize:          *bufferSize,
		LocalAddress:        *localAddr,
		RemoteAddress:       *remoteAddr,
		ServerHost:          *serverHost,
		DisableServerResolv: *disableServerResolv,
		LocalPayload:        *localPayload,
		RemotePayload:       *remotePayload,
		TLSEnabled:          *tlsEnabled,
		SNIHost:             *sniHost,
	}
	common.ParseConfig(config, *configFile, cmdArgs)

	listener, err := net.Listen("tcp", config.LocalAddressTCP.String())
	if err != nil {
		fmt.Printf("Failed to open local port to listen: %s", err)
		return
	}

	fmt.Printf("Mode\t\t: %s\n", config.ProxyInfo)
	fmt.Printf("Buffer size\t: %d\n", config.BufferSize)
	fmt.Printf("Connection\t: %s\n", config.ConnectionInfo)
	if config.TLSEnabled {
		fmt.Printf("SNI Host\t: %s\n", config.SNIHost)
	}
	fmt.Printf("\ngo-tcp-proxy-tunnel proxing from %v to %v\n", config.LocalAddressTCP, config.RemoteAddressTCP)

	handleListener(listener, config)
}

func handleListener(listener net.Listener, config *common.Config) {
	var connId = uint64(0)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection '%s'", err)
			return
		}
		connId += 1

		p := proxy.NewProxy(connId, conn, config.LocalAddressTCP, config.RemoteAddressTCP)
		if config.ServerHost != "" {
			p.SetServerHost(config.ServerHost)
		}
		if config.BufferSize > 0 {
			p.SetBufferSize(config.BufferSize)
		}
		if config.TLSEnabled {
			p.SetEnableTLS(config.TLSEnabled)
			p.SetSNIHost(config.SNIHost)
		}
		p.SetlPayload(config.LocalPayload)
		p.SetrPayload(config.RemotePayload)
		p.SetServerProxyMode(config.ServerProxyMode)
		go p.Start()
	}
}
