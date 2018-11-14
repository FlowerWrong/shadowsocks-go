package shadowsocks

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/FlowerWrong/shadowsocks-go/socks"
	ssUtil "github.com/FlowerWrong/shadowsocks-go/util"
	"github.com/FlowerWrong/util"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

func handleConnection(conn net.Conn, shadow func(net.Conn) net.Conn, ss string) {
	defer conn.Close()

	err := socks.HandleConnectAndAuth(conn)
	if err != nil {
		log.Println(err)
		return
	}
	socks5Req, err := socks.HandleRequest(conn)
	if err != nil {
		log.Println(err)
		return
	}

	remoteConn, err := net.Dial("tcp", ss)
	if err != nil {
		log.Println(err)
		return
	}
	remoteConn = shadow(remoteConn)

	// https://shadowsocks.org/en/spec/Protocol.html
	// [1-byte type][variable-length host][2-byte port]
	if _, err = remoteConn.Write(socks5Req.Tgt); err != nil {
		log.Println(err)
		return
	}

	host, port := socks.HostPort(socks5Req)
	log.Printf("proxy %s <-> %s <-> %s", conn.RemoteAddr(), ss, net.JoinHostPort(host, port))

	go io.Copy(remoteConn, conn)
	io.Copy(conn, remoteConn)
}

// Serve ...
func Serve(serverURLs util.ArrayFlags) {
	wgw := new(util.WaitGroupWrapper)

	for _, ss := range serverURLs {
		host, method, password, localPort, err := ssUtil.ParseSSURL(ss)
		if err != nil {
			log.Fatal(err)
		}

		wgw.Wrap(func() {
			ciph, err := core.PickCipher(method, []byte{}, password)
			if err != nil {
				log.Fatal(err)
			}

			ln, err := net.Listen("tcp", fmt.Sprintf(":%s", localPort))
			if err != nil {
				log.Fatalln(err)
			}
			for {
				conn, err := ln.Accept()
				if err != nil {
					log.Println("Accept failed", err)
					continue
				}
				go handleConnection(conn, ciph.StreamConn, host)
			}
		})
	}

	wgw.WaitGroup.Wait()
}
