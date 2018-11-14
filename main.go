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
// go run main.go --server '' --server ''
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	var serverURLs util.ArrayFlags
	flag.Var(&serverURLs, "server", "ss server URL")
	flag.Parse()
	shadowsocks.Serve(serverURLs)
}
