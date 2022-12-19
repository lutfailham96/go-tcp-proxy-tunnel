package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/common"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/logger"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/util"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/pkg/proxy"
	"net"
	"os"
	"path/filepath"
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
	tlsCert             = flag.String("cert", "", "tls cert pem file")
	tlsKey              = flag.String("key", "", "tls key pem file")
	proxyKind           = flag.String("k", "ssh", "proxy kind [ssh, trojan] (default: ssh)")
	logLevel            = flag.Uint64("lv", 3, "log level [1-5]")
)

func main() {
	flag.Parse()

	log := logger.NewBaseLogger(logger.LogLevel(*logLevel))

	cmdArgs := &common.CmdArgs{
		LocalAddress:        *localAddr,
		RemoteAddress:       *remoteAddr,
		ServerHost:          *serverHost,
		DisableServerResolv: *disableServerResolv,
		ProxyKind:           *proxyKind,
	}

	config := &common.Config{
		ServerProxyMode:     *serverProxyMode,
		ProxyKind:           *proxyKind,
		BufferSize:          *bufferSize,
		LocalAddress:        *localAddr,
		RemoteAddress:       *remoteAddr,
		ServerHost:          *serverHost,
		DisableServerResolv: *disableServerResolv,
		LocalPayload:        *localPayload,
		RemotePayload:       *remotePayload,
		TLSEnabled:          *tlsEnabled,
		TLSCert:             *tlsCert,
		TLSKey:              *tlsKey,
		SNIHost:             *sniHost,
	}
	common.ParseConfig(config, *configFile, cmdArgs)

	var listener net.Listener
	var err error
	if config.TLSEnabled && config.ProxyKind != "ssh" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         config.SNIHost,
		}
		if config.TLSCert != "" || config.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
			if err != nil {
				log.PrintCritical(fmt.Sprintf("Cannot load tls key pair '%s'\n", err))
				return
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		} else {
			ex, _ := os.Executable()
			exDir := filepath.Dir(ex)
			crtPath := fmt.Sprintf("%s/server.crt", exDir)
			keyPath := fmt.Sprintf("%s/server.key", exDir)
			_, errCrt := os.Stat(crtPath)
			_, errKey := os.Stat(keyPath)
			if errCrt == nil || errKey == nil {
				cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
				if err != nil {
					log.PrintCritical(fmt.Sprintf("Cannot read tls key pair '%s'\n", err))
					os.Exit(1)
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
			} else {
				serverConfig, _, err := util.TLSGenerateConfig()
				if err != nil {
					log.PrintCritical(fmt.Sprintf("Cannot generate tls key pair '%s'\n", err))
					return
				}
				tlsConfig.Certificates = serverConfig.Certificates
				// TODO write generated cert & private key to `server.crt`, `server.key`
			}
		}
		listener, err = tls.Listen("tcp", config.LocalAddressTCP.String(), tlsConfig)
		if err != nil {
			log.PrintCritical(fmt.Sprintf("Failed to open local port to listen: %s\n", err))
			return
		}
	} else {
		listener, err = net.Listen("tcp", config.LocalAddressTCP.String())
	}
	if err != nil {
		log.PrintCritical(fmt.Sprintf("Failed to open local port to listen: %s\n", err))
		return
	}

	log.PrintInfo(fmt.Sprintf("Mode\t\t: %s\n", config.ProxyInfo))
	log.PrintInfo(fmt.Sprintf("Proxy Kind\t: %s\n", config.ProxyKind))
	log.PrintInfo(fmt.Sprintf("Buffer size\t: %d\n", config.BufferSize))
	log.PrintInfo(fmt.Sprintf("Connection\t: %s\n", config.ConnectionInfo))
	log.PrintInfo(fmt.Sprintf("SNI Host\t: %s\n", config.SNIHost))
	log.PrintInfo(fmt.Sprintf("\ngo-tcp-proxy-tunnel proxing from %v to %v\n", config.LocalAddressTCP, config.RemoteAddressTCP))

	handleListener(listener, config, log)
}

func handleListener(listener net.Listener, config *common.Config, log *logger.BaseLogger) {
	var connId = uint64(0)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.PrintError(fmt.Sprintf("Failed to accept connection '%s'\n", err))
			return
		}
		connId += 1

		p := proxy.NewProxy(connId, conn, config.LocalAddressTCP, config.RemoteAddressTCP, config.TLSEnabled)
		if config.ServerHost != "" {
			p.SetServerHost(config.ServerHost)
		}
		if config.BufferSize > 0 {
			p.SetBufferSize(config.BufferSize)
		}
		if config.TLSEnabled {
			p.SetEnableTLS(config.TLSEnabled)
		}
		p.SetSNIHost(config.SNIHost)
		p.SetlPayload(config.LocalPayload)
		p.SetrPayload(config.RemotePayload)
		p.SetServerProxyMode(config.ServerProxyMode)
		p.SetProxyKind(config.ProxyKind)
		p.SetLogger(log)
		go p.Start()
	}
}
