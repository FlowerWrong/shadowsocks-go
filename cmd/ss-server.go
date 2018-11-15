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

	wgw := new(util.WaitGroupWrapper)
	wgw.Wrap(func() {
		shadowsocks.ServeRemoteTCP(serverURLs)
	})
	wgw.Wrap(func() {
		shadowsocks.ServeRemoteUDP(serverURLs)
	})
	wgw.WaitGroup.Wait()
}
