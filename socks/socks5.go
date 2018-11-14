// Package socks 5 rfc https://datatracker.ietf.org/doc/rfc1928/
package socks

import (
	"errors"
	"io"
	"net"
	"strconv"

	socks5 "github.com/shadowsocks/go-shadowsocks2/socks"
)

const maxLen = 1 + 1 + 255

// VER is supported socks version
const VER = 0x05

// CMDxxx is socks 5 Cmd
const (
	CMDCONNECT = 0x01
	CMDBIND    = 0x02
	CMDUDP     = 0x03
)

// ATYPExxx is socks 5 aype
const (
	ATYPEIPv4   = 0x01
	ATYPEDOMAIN = 0x03
	ATYPEIPv6   = 0x04
)

// Request ...
type Request struct {
	Cmd     byte
	Atype   byte
	DstAddr []byte
	DstPort int
	Tgt     []byte
}

// ParseUDPRequest ... See https://www.ietf.org/rfc/rfc1928.txt
func ParseUDPRequest(buf []byte) *Request {
	// +----+------+------+----------+----------+----------+
	// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
	// +----+------+------+----------+----------+----------+
	// | 2  |  1   |  1   | Variable |    2     | Variable |
	// +----+------+------+----------+----------+----------+
	atype := buf[3]
	req := &Request{Cmd: CMDUDP, Atype: atype}
	req.Tgt = []byte{atype}
	if atype == ATYPEIPv4 {
		req.DstAddr = buf[4 : 4+net.IPv4len]
		req.DstPort = int(buf[4+net.IPv4len])<<8 | int(buf[4+net.IPv4len+1])
		req.Tgt = append(req.Tgt, buf[4:4+net.IPv4len+2]...)
	} else if atype == ATYPEDOMAIN {
		req.Tgt = append(req.Tgt, buf[5])
		domainLen := buf[5]
		req.DstAddr = buf[5 : 5+domainLen]
		req.DstPort = int(buf[5+domainLen])<<8 | int(buf[5+domainLen+1])
		req.Tgt = append(req.Tgt, buf[5:5+domainLen+2]...)
	} else if atype == ATYPEIPv6 {
		req.DstAddr = buf[4 : 4+net.IPv6len]
		req.DstPort = int(buf[4+net.IPv6len])<<8 | int(buf[4+net.IPv6len+1])
		req.Tgt = append(req.Tgt, buf[4:4+net.IPv6len+2]...)
	}
	return req
}

// HostPort ...
func HostPort(socks5Req *Request) (host, port string) {
	port = strconv.Itoa(socks5Req.DstPort)
	if socks5Req.Atype == ATYPEIPv4 || socks5Req.Atype == ATYPEIPv6 {
		host = net.IP(socks5Req.DstAddr[:]).String()
	} else if socks5Req.Atype == ATYPEDOMAIN {
		host = string(socks5Req.DstAddr[:])
	}
	return
}

// HandleConnectAndAuth ...
func HandleConnectAndAuth(rw io.ReadWriter) error {
	var buf [maxLen]byte
	// VER NMETHODS
	_, err := io.ReadFull(rw, buf[:2])
	if err != nil {
		return err
	}

	// check socks 5
	if buf[0] != VER {
		return errors.New("only support socks 5")
	}

	nmethods := buf[1]
	_, err = io.ReadFull(rw, buf[:nmethods])
	if err != nil {
		return err
	}

	_, err = rw.Write([]byte{VER, 0x00}) // no auth
	if err != nil {
		return err
	}

	return nil
}

// HandleRequest ...
func HandleRequest(rw io.ReadWriter) (*Request, error) {
	var buf [maxLen]byte
	// VER CMD RSV ATYP
	_, err := io.ReadFull(rw, buf[:4])
	if err != nil {
		return nil, err
	}

	// check socks 5
	if buf[0] != VER {
		return nil, errors.New("only support socks 5")
	}

	cmd := buf[1]
	atype := buf[3]

	req := &Request{Cmd: cmd, Atype: atype}
	req.Tgt = []byte{atype}

	if atype == ATYPEIPv4 {
		_, err = io.ReadFull(rw, buf[:net.IPv4len+2])
		if err != nil {
			return nil, err
		}
		req.DstAddr = buf[:net.IPv4len]
		req.DstPort = int(buf[net.IPv4len])<<8 | int(buf[net.IPv4len+1])
		req.Tgt = append(req.Tgt, buf[:net.IPv4len+2]...)
	} else if atype == ATYPEDOMAIN {
		_, err = io.ReadFull(rw, buf[:1])
		if err != nil {
			return nil, err
		}
		req.Tgt = append(req.Tgt, buf[0])
		domainLen := buf[0]
		_, err = io.ReadFull(rw, buf[:domainLen+2])
		if err != nil {
			return nil, err
		}
		req.DstAddr = buf[:domainLen]
		req.DstPort = int(buf[domainLen])<<8 | int(buf[domainLen+1])
		req.Tgt = append(req.Tgt, buf[:domainLen+2]...)
	} else if atype == ATYPEIPv6 {
		_, err = io.ReadFull(rw, buf[:net.IPv6len+2])
		if err != nil {
			return nil, err
		}
		req.DstAddr = buf[:net.IPv6len]
		req.DstPort = int(buf[net.IPv6len])<<8 | int(buf[net.IPv6len+1])
		req.Tgt = append(req.Tgt, buf[:net.IPv6len+2]...)
	} else {
		return nil, errors.New("invalid Cmd")
	}

	if cmd == CMDCONNECT {
		// connect will ignore BND.ADDR and BND.PORT
		_, err = rw.Write([]byte{VER, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		if err != nil {
			return nil, err
		}
	} else if cmd == CMDUDP {
		// In the reply to a UDP ASSOCIATE request, the BND.PORT and BND.ADDR
		// fields indicate the port number/address where the client MUST send
		// UDP request messages to be relayed. @see https://www.ietf.org/rfc/rfc1928.txt
		listenAddr := socks5.ParseAddr(rw.(net.Conn).LocalAddr().String())
		_, err = rw.Write(append([]byte{VER, 0x00, 0x00}, listenAddr...))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("only support connect and udp")
	}

	return req, nil
}
