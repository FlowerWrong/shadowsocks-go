// https://github.com/songgao/water#tun-on-macos
package main

import (
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/FlowerWrong/kone/tcpip"
	"github.com/songgao/water"
)

// NOTE: below command is just for OSX.
// sudo go run cmd/utun.go
// sudo ifconfig utunx 10.1.0.10 10.1.0.20 up
// netstat -rn | grep utunx
// ping 10.1.0.20
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 创建tun设备
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Interface Name: %s\n", ifce.Name())

	b := make([]byte, 1500)
	for {
		n, err := ifce.Read(b)
		if err != nil {
			log.Fatal(err)
		}

		packet := b[:n]
		if tcpip.IsIPv4(packet) {
			ipPacket := tcpip.IPv4Packet(packet)
			icmpPacket := tcpip.ICMPPacket(ipPacket.Payload())
			if icmpPacket.Type() == tcpip.ICMPRequest && icmpPacket.Code() == 0 {
				log.Printf("icmp echo request: %s -> %s\n", ipPacket.SourceIP(), ipPacket.DestinationIP())
				// 如果是icmp request，就构造一个icmp response，然后添加ip头部
				icmpPacket.SetType(tcpip.ICMPEcho)
				srcIP := ipPacket.SourceIP()
				dstIP := ipPacket.DestinationIP()
				ipPacket.SetSourceIP(dstIP)
				ipPacket.SetDestinationIP(srcIP)

				icmpPacket.ResetChecksum()
				ipPacket.ResetChecksum()

				// 写会tun设备，这里是ip数据包，如果是tap，那么就是帧，例如以太网帧
				ifce.Write(ipPacket)
			} else {
				log.Printf("icmp: %s -> %s\n", ipPacket.SourceIP(), ipPacket.DestinationIP())
			}
		}
	}
}
