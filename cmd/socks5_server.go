package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"time"

	"github.com/FlowerWrong/shadowsocks-go/shadowsocks"
	"github.com/FlowerWrong/shadowsocks-go/socks"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 连接认证阶段
	err := socks.HandleConnectAndAuth(conn)
	if err != nil {
		log.Println(err)
		return
	}

	// 请求阶段
	socks5Req, err := socks.HandleRequest(conn)
	if err != nil {
		log.Println(err)
		return
	}

	host, port := socks.HostPort(socks5Req)
	log.Printf("proxy %s <-> %s", conn.RemoteAddr(), net.JoinHostPort(host, port))

	d := net.Dialer{Control: shadowsocks.SetSocketOptions}
	remoteConn, err := d.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Println(err)
		return
	}

	// 数据传输阶段，对倒tcp flow
	go io.Copy(remoteConn, conn)
	io.Copy(conn, remoteConn)
}

// go run cmd/socks5_server.go
// curl -v --socks5-hostname 127.0.0.1:2090 baidu.com
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	lc := net.ListenConfig{Control: shadowsocks.SetSocketOptions}
	ln, err := lc.Listen(context.Background(), "tcp", ":2090")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("socks 5 server start on %s", ln.Addr().String())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept failed", err)
			continue
		}
		go handleConnection(conn)
	}
}
