package main

import (
	"encoding/json"
	"flag"
	"fmt"
	proxy "github.com/lutfailham96/go-tcp-proxy-tunnel"
	"net"
	"os"
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

	config := &proxy.Config{
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
	parseConfig(config, *configFile)

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

	loopListener(listener, config)
}

func resolveAddr(addr string) *net.TCPAddr {
	if addr == "" {
		fmt.Println("Host address is not valid or empty")
		os.Exit(1)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to resolve local address: %s", err)
		os.Exit(1)
	}
	return tcpAddr
}

func loopListener(listener net.Listener, config *proxy.Config) {
	var connId = uint64(0)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection '%s'", err)
			return
		}
		connId += 1

		var p *proxy.Proxy
		p = p.New(connId, conn, config.LocalAddressTCP, config.RemoteAddressTCP)
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

func parseConfig(config *proxy.Config, configFile string) {
	if configFile != "" {
		file, err := os.Open(configFile)
		if err != nil {
			fmt.Printf("Cannot open file '%s'", err)
			os.Exit(1)
			return
		}
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				fmt.Printf("Cannot close file '%s", err)
				return
			}
		}(file)

		jsonDecoder := json.NewDecoder(file)
		err = jsonDecoder.Decode(config)
		if err != nil {
			fmt.Printf("Cannot decode config file '%s'", err)
			os.Exit(1)
			return
		}
	}

	localAddress := *localAddr
	if config.LocalAddress != "" {
		localAddress = config.LocalAddress
	}
	config.LocalAddressTCP = resolveAddr(localAddress)

	remoteAddress := *remoteAddr
	if config.RemoteAddress != "" {
		remoteAddress = config.RemoteAddress
	}
	config.RemoteAddressTCP = resolveAddr(remoteAddress)

	serverHostAddr := *serverHost
	if config.ServerHost != "" {
		serverHostAddr = config.ServerHost
	}
	if serverHostAddr != "" && !*disableServerResolv {
		resolveAddr(serverHostAddr)
	}

	config.ConnectionInfo = "insecure"
	if config.TLSEnabled {
		if config.SNIHost == "" {
			fmt.Println("SNI hostname required on secure connection")
			os.Exit(1)
			return
		}
		config.ConnectionInfo = "secure (TLS)"
	}

	if config.BufferSize == 0 {
		config.BufferSize = 0xffff
	}

	config.ProxyInfo = "client proxy"
	if config.ServerProxyMode {
		config.ProxyInfo = "server proxy"
	}
}
