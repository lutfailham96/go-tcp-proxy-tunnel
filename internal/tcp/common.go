package tcp

import (
	"fmt"
	"net"
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
