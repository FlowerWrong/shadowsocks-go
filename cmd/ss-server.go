package main

import (
	"flag"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/FlowerWrong/shadowsocks-go/shadowsocks"
	"github.com/FlowerWrong/util"
)

// ss://method:password@:port
// go run ss-server.go --server '' --server ''
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	var serverURLs util.ArrayFlags
	flag.Var(&serverURLs, "server", "ss server URL")
	flag.Parse()

	// See https://golang.org/pkg/sync/#WaitGroup
	wgw := new(util.WaitGroupWrapper)
	wgw.Wrap(func() {
		// 处理tcp连接，其实就是一个socks 5 proxy server
		shadowsocks.ServeRemoteTCP(serverURLs)
	})
	wgw.Wrap(func() {
		// 处理udp连接，其实就是一个socks 5 proxy server
		shadowsocks.ServeRemoteUDP(serverURLs)
	})
	wgw.WaitGroup.Wait()
}
