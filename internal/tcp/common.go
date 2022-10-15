package tcp

import (
	"fmt"
	"net"
	"os"
)

type Host struct {
	HostName string
	Port     uint64
}

func CloseConnection(conn net.Conn) {
	err := conn.Close()
	if err != nil {
		fmt.Printf("Cannot close connection '%s'", err)
		return
	}
}

func ResolveAddr(addr string) *net.TCPAddr {
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
