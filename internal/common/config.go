package common

import (
	"net"
)

type Config struct {
	BufferSize          uint64
	ServerProxyMode     bool
	ProxyInfo           string
	LocalAddress        string
	RemoteAddress       string
	LocalAddressTCP     *net.TCPAddr
	RemoteAddressTCP    *net.TCPAddr
	ServerHost          string
	DisableServerResolv bool
	ConnectionInfo      string
	TLSEnabled          bool
	SNIHost             string
	LocalPayload        string
	RemotePayload       string
}
