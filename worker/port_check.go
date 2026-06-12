package worker

import (
	"fmt"
	"net"
	"time"
)

// IsPortOpen checks if a TCP port is accepting connections.
func IsPortOpen(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}