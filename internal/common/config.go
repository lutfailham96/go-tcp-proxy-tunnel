package common

import (
	"encoding/json"
	"fmt"
	"github.com/lutfailham96/go-tcp-proxy-tunnel/internal/tcp"
	"net"
	"os"
)

type Config struct {
	BufferSize          uint64
	ServerProxyMode     bool
	ProxyKind           string
	ProxyInfo           string
	LocalAddress        string
	RemoteAddress       string
	LocalAddressTCP     *net.TCPAddr
	RemoteAddressTCP    *net.TCPAddr
	ServerHost          string
	DisableServerResolv bool
	ConnectionInfo      string
	TLSEnabled          bool
	TLSCert             string
	TLSKey              string
	SNIHost             string
	LocalPayload        string
	RemotePayload       string
}

type CmdArgs struct {
	LocalAddress        string
	RemoteAddress       string
	ServerHost          string
	DisableServerResolv bool
	ProxyKind           string
}

func (cfg *Config) setDefaults() {
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 0xffff
	}

	cfg.ProxyInfo = "client proxy"
	if cfg.ServerProxyMode {
		cfg.ProxyInfo = "server proxy"
	}
}

func ParseConfig(config *Config, configFile string, cmdArgs *CmdArgs) {
	loadConfigFile(configFile, config)

	localAddress := cmdArgs.LocalAddress
	if config.LocalAddress != "" {
		localAddress = config.LocalAddress
	}
	config.LocalAddressTCP = tcp.ResolveAddr(localAddress)

	remoteAddress := cmdArgs.RemoteAddress
	if config.RemoteAddress != "" {
		remoteAddress = config.RemoteAddress
	}
	config.RemoteAddressTCP = tcp.ResolveAddr(remoteAddress)

	serverHostAddr := cmdArgs.ServerHost
	if config.ServerHost != "" {
		serverHostAddr = config.ServerHost
	}
	if serverHostAddr != "" && !cmdArgs.DisableServerResolv {
		tcp.ResolveAddr(serverHostAddr)
	}

	config.ConnectionInfo = "insecure"
	if config.TLSEnabled {
		if config.SNIHost == "" {
			fmt.Printf("SNI hostname required on secure connection\n")
			os.Exit(1)
			return
		}
		config.ConnectionInfo = "secure (TLS)"
	}

	config.setDefaults()
}

func loadConfigFile(cfgFile string, cfg *Config) {
	if cfgFile != "" {
		file, err := os.Open(cfgFile)
		if err != nil {
			fmt.Printf("Cannot open file '%s'\n", err)
			os.Exit(1)
			return
		}
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				fmt.Printf("Cannot close file '%s'\n", err)
				return
			}
		}(file)

		jsonDecoder := json.NewDecoder(file)
		err = jsonDecoder.Decode(cfg)
		if err != nil {
			fmt.Printf("Cannot decode config file '%s'\n", err)
			os.Exit(1)
			return
		}
	}
}
