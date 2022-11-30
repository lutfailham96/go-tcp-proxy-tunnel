package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/common"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/tcp"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/util"
	"net"
	"os"
	"path/filepath"
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
	logLevel       = flag.Uint64("lv", 3, "log level")
)

func main() {
	flag.Parse()

	logger := common.NewBaseLogger(common.LogLevel(*logLevel))
	logger.PrintInfo(fmt.Sprintf("SNI:\t\t\t%s\n", *sni))

	if *sni == "" {
		logger.PrintCritical(fmt.Sprintf("SNI required!"))
		os.Exit(1)
	}

	var tcpWg sync.WaitGroup

	tcpWg.Add(2)
	go setupTcpListener(false, logger)
	go setupTcpListener(true, logger)

	tcpWg.Wait()
}

func setupTcpListener(secure bool, log *common.BaseLogger) {
	var ln net.Listener
	var err error

	if secure {
		var tlsConfig *tls.Config
		if *tlsCert != "" && *tlsKey != "" {
			cer, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
			if err != nil {
				log.PrintCritical(fmt.Sprintf("Cannot read tls key pair '%s'\n", err))
			}
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cer},
			}
		} else {
			ex, err := os.Executable()
			if err != nil {
				panic(err)
			}
			exDir := filepath.Dir(ex)
			crtPath := fmt.Sprintf("%s/server.crt", exDir)
			keyPath := fmt.Sprintf("%s/server.key", exDir)
			_, errCrt := os.Stat(crtPath)
			_, errKey := os.Stat(keyPath)
			if errCrt == nil && errKey == nil {
				cer, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
				if err != nil {
					log.PrintCritical(fmt.Sprintf("Cannot read tls key pair '%s'\n", err))
					os.Exit(1)
				}
				tlsConfig = &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cer},
				}
			} else {
				tlsConfig, _, err = util.TLSGenerateConfig()
				if err != nil {
					log.PrintCritical(fmt.Sprintf("Cannot setup tls certificates '%s'\n", err))
				}
				// TODO write generated cert & private key to `server.crt`, `server.key`
			}
		}
		tcp.ResolveAddr(*httpsAddress)
		ln, err = tls.Listen("tcp", *httpsAddress, tlsConfig)
		log.PrintInfo(fmt.Sprintf("Secure TCP listen on:\t%s\n", *httpsAddress))
	} else {
		tcp.ResolveAddr(*httpAddress)
		ln, err = net.Listen("tcp", *httpAddress)
		log.PrintInfo(fmt.Sprintf("TCP listen on:\t\t%s\n", *httpAddress))
	}
	if err != nil {
		log.PrintCritical(fmt.Sprintf("Cannot bind port '%s'\n", err))
		return
	}

	connId := uint64(0)
	for {
		src, err := ln.Accept()
		if err != nil {
			log.PrintError(fmt.Sprintf("Cannot accept connection '%s'\n", err))
			continue
		}
		connId += 1
		fwd := tcp.NewWebForwarder(connId, src, secure)
		fwd.SetDstAddress(*backendAddress)
		fwd.SetTrjConfig(*trojanAddress, *trojanWsPath)
		fwd.SetSNI(*sni)
		fwd.SetLogger(log)
		go fwd.Start()
	}
}
