package main

import (
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"strconv"
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
	log.Println(socks5Req)

	port := strconv.Itoa(socks5Req.DstPort)
	var host string
	if socks5Req.Atype == socks.ATYPEIPv4 || socks5Req.Atype == socks.ATYPEIPv6 {
		host = net.IP(socks5Req.DstAddr[:]).String()
	} else if socks5Req.Atype == socks.ATYPEDOMAIN {
		host = string(socks5Req.DstAddr[:])
	}

	remoteConn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Println(err)
		return
	}
	go io.Copy(remoteConn, conn)
	io.Copy(conn, remoteConn)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	ln, err := net.Listen("tcp", ":1090")
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
