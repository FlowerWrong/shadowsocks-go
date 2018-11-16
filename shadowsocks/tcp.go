package shadowsocks

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/FlowerWrong/shadowsocks-go/socks"
	ssUtil "github.com/FlowerWrong/shadowsocks-go/util"
	"github.com/FlowerWrong/util"
	"github.com/shadowsocks/go-shadowsocks2/core"
	socks5 "github.com/shadowsocks/go-shadowsocks2/socks"
)

// MTU ...
const MTU = 1500

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

	// 如果是UDP请求，那么这里tcp连接不能关闭，tcp和udp在socks5里面是一一对应的关系
	// tcp断了，那么代理通道就断了
	if socks5Req.Cmd == socks.CMDUDP {
		var buf [MTU]byte
		// block until close
		for {
			_, err = conn.Read(buf[:])
			if err != nil {
				if !util.IsEOF(err) && !util.IsTimeout(err) {
					log.Println(err)
				}
				return
			}
		}
	}

	remoteConn, err := net.Dial("tcp", ss)
	if err != nil {
		log.Println(err)
		return
	}
	defer remoteConn.Close()
	remoteConn = shadow(remoteConn)

	// https://shadowsocks.org/en/spec/Protocol.html
	// [1-byte type][variable-length host][2-byte port]
	if _, err = remoteConn.Write(socks5Req.Tgt); err != nil {
		log.Println(err)
		return
	}

	host, port := socks.HostPort(socks5Req)
	log.Printf("proxy %s <-> %s <-> %s", conn.RemoteAddr(), ss, net.JoinHostPort(host, port))

	// 对倒数据
	go io.Copy(remoteConn, conn)
	io.Copy(conn, remoteConn)
}

// ServeTCP  is for ss-local
func ServeTCP(serverURLs util.ArrayFlags) {
	var wg sync.WaitGroup

	for i, ss := range serverURLs {
		wg.Add(1)
		go func(i int, ss string) {
			defer wg.Done()
			host, method, password, localPort, err := ssUtil.ParseSSURL(ss)
			if err != nil {
				log.Fatal(err)
			}

			cipher, err := core.PickCipher(method, []byte{}, password)
			if err != nil {
				log.Fatal(err)
			}

			ln, err := net.Listen("tcp", fmt.Sprintf(":%s", localPort))
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("ss TCP server %d start on %s", i, ln.Addr().String())
			for {
				conn, err := ln.Accept()
				if err != nil {
					log.Println("Accept failed", err)
					continue
				}

				// cipher.StreamConn 可以想象成安全层和应用层的关系，类似http + tls
				go handleConnection(conn, cipher.StreamConn, host)
			}
		}(i, ss)
	}

	wg.Wait()
}

// ServeRemoteTCP  is for ss-server
func ServeRemoteTCP(serverURLs util.ArrayFlags) {
	var wg sync.WaitGroup

	for i, ss := range serverURLs {
		wg.Add(1)
		go func(i int, ss string) {
			defer wg.Done()
			host, method, password, _, err := ssUtil.ParseSSURL(ss)
			if err != nil {
				log.Fatal(err)
			}

			cipher, err := core.PickCipher(method, []byte{}, password)
			if err != nil {
				log.Fatal(err)
			}

			ln, err := net.Listen("tcp", host)
			if err != nil {
				log.Fatalln(err)
			}
			log.Printf("ss TCP server %d start on %s", i, ln.Addr().String())
			for {
				conn, err := ln.Accept()
				if err != nil {
					log.Println("Accept failed", err)
					continue
				}
				go func() {
					defer conn.Close()
					// cipher.StreamConn 可以想象成安全层和应用层的关系，类似http + tls
					conn = cipher.StreamConn(conn)

					tgt, err := socks5.ReadAddr(conn)
					if err != nil {
						log.Println(err)
						return
					}

					remoteConn, err := net.Dial("tcp", tgt.String())
					if err != nil {
						log.Println(err)
						return
					}
					defer remoteConn.Close()

					log.Printf("proxy %s <-> %s", conn.RemoteAddr(), tgt)

					// 对倒数据
					go io.Copy(remoteConn, conn)
					io.Copy(conn, remoteConn)
				}()
			}
		}(i, ss)
	}

	wg.Wait()
}
