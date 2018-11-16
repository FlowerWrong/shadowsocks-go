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

// curl -v --socks5-hostname 127.0.0.1:1080 baidu.com
// ss://method:password@hostname:port?local_port=1080
// go run ss-local.go --server '' --server ''
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
		shadowsocks.ServeTCP(serverURLs)
	})
	wgw.Wrap(func() {
		// 处理udp连接，其实就是一个socks 5 proxy server
		shadowsocks.ServeUDP(serverURLs)
	})
	wgw.WaitGroup.Wait()
}
