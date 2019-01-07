package shadowsocks

import (
	"log"
	"syscall"
)

// SetSocketOptions functions sets IP_TRANSPARENT flag on given socket (c syscall.RawConn)
func SetSocketOptions(network, address string, c syscall.RawConn) error {
	var fn = func(fd uintptr) {
		log.Printf("socket fd is: %d", fd)
	}
	if err := c.Control(fn); err != nil {
		return err
	}

	return nil
}
