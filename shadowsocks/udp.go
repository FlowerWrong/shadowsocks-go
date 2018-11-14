package shadowsocks

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/FlowerWrong/shadowsocks-go/socks"
	ssUtil "github.com/FlowerWrong/shadowsocks-go/util"
	"github.com/FlowerWrong/util"
	"github.com/shadowsocks/go-shadowsocks2/core"
)

// MAXUDPPACKETSIZE ...
const MAXUDPPACKETSIZE = 65507

func readFromRemoteWriteToLocal(remotePC, localPC net.PacketConn, localAddr net.Addr) {
	remoteBuf := make([]byte, MAXUDPPACKETSIZE)
	for {
		// The reassembly timer MUST be no less than 5 seconds?
		remotePC.SetReadDeadline(time.Now().Add(time.Second * 5))
		m, _, err := remotePC.ReadFrom(remoteBuf)
		if err != nil {
			if !util.IsEOF(err) && !util.IsTimeout(err) {
				log.Println(err)
			}
			break
		}

		_, err = localPC.WriteTo(append([]byte{0, 0, 0}, remoteBuf[:m]...), localAddr)
		if err != nil {
			log.Println(err)
			break
		}
	}
}

// ServeUDP ...
func ServeUDP(serverURLs util.ArrayFlags) {
	var wg sync.WaitGroup

	for i, ss := range serverURLs {
		wg.Add(1)
		go func(i int, ss string) {
			defer wg.Done()
			host, method, password, localPort, err := ssUtil.ParseSSURL(ss)
			if err != nil {
				log.Fatal(err)
			}

			ssAddr, err := net.ResolveUDPAddr("udp", host)
			if err != nil {
				log.Fatal(err)
			}

			cipher, err := core.PickCipher(method, []byte{}, password)
			if err != nil {
				log.Fatal(err)
			}

			localPC, err := net.ListenPacket("udp", fmt.Sprintf(":%s", localPort))
			if err != nil {
				log.Fatal(err)
			}
			defer localPC.Close()
			log.Printf("ss UDP server %d start on %s", i, localPC.LocalAddr().String())

			buf := make([]byte, MAXUDPPACKETSIZE)
			for {
				n, remoteAddr, err := localPC.ReadFrom(buf)
				if err != nil {
					log.Println(err)
					continue
				}

				remotePC, err := net.ListenPacket("udp", "")
				if err != nil {
					log.Println(err)
					continue
				}
				defer remotePC.Close()
				remotePC = cipher.PacketConn(remotePC)

				socks5Req := socks.ParseUDPRequest(buf[:n])
				targetHost, targetPort := socks.HostPort(socks5Req)
				log.Printf("proxy %s <-> %s <-> %s", remoteAddr, host, net.JoinHostPort(targetHost, targetPort))
				go readFromRemoteWriteToLocal(remotePC, localPC, remoteAddr)
				_, err = remotePC.WriteTo(buf[3:n], ssAddr)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}(i, ss)
	}

	wg.Wait()
}
