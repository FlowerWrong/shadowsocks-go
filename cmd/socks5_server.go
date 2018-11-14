package main

import (
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"time"

	"github.com/FlowerWrong/shadowsocks-go/socks"
)

func handleConnection(conn net.Conn) {
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

	host, port := socks.HostPort(socks5Req)
	log.Printf("proxy %s <-> %s", conn.RemoteAddr(), net.JoinHostPort(host, port))

	remoteConn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Println(err)
		return
	}
	go io.Copy(remoteConn, conn)
	io.Copy(conn, remoteConn)
}

// curl -v --socks5-hostname 127.0.0.1:2090 baidu.com
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	ln, err := net.Listen("tcp", ":2090")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept failed", err)
			continue
		}
		go handleConnection(conn)
	}
}
