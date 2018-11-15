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
	socks5 "github.com/shadowsocks/go-shadowsocks2/socks"
)

// MAXUDPPACKETSIZE ...
const MAXUDPPACKETSIZE = 65507

func readFromRemoteWriteToLocal(remotePC, localPC net.PacketConn, localAddr net.Addr) {
	defer remotePC.Close()
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

		// +----+------+------+----------+----------+----------+
		// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
		// +----+------+------+----------+----------+----------+
		// | 2  |  1   |  1   | Variable |    2     | Variable |
		// +----+------+------+----------+----------+----------+
		_, err = localPC.WriteTo(append([]byte{0, 0, 0}, remoteBuf[:m]...), localAddr)
		if err != nil {
			log.Println(err)
			break
		}
	}
}

// ServeUDP is for ss-local
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
				remotePC = cipher.PacketConn(remotePC)

				socks5Req := socks.ParseUDPRequest(buf[:n])
				targetHost, targetPort := socks.HostPort(socks5Req)
				log.Printf("proxy %s <-> %s <-> %s", remoteAddr, host, net.JoinHostPort(targetHost, targetPort))
				go readFromRemoteWriteToLocal(remotePC, localPC, remoteAddr)

				// +------+----------+----------+----------+
				// | ATYP | DST.ADDR | DST.PORT |   DATA   |
				// +------+----------+----------+----------+
				// |  1   | Variable |    2     | Variable |
				// +------+----------+----------+----------+
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

// ServeRemoteUDP is for ss-srever
func ServeRemoteUDP(serverURLs util.ArrayFlags) {
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

			localPC, err := net.ListenPacket("udp", host)
			if err != nil {
				log.Fatal(err)
			}
			defer localPC.Close()
			localPC = cipher.PacketConn(localPC)
			log.Printf("ss UDP server %d start on %s", i, localPC.LocalAddr().String())

			buf := make([]byte, MAXUDPPACKETSIZE)
			for {
				// +------+----------+----------+----------+
				// | ATYP | DST.ADDR | DST.PORT |   DATA   |
				// +------+----------+----------+----------+
				// |  1   | Variable |    2     | Variable |
				// +------+----------+----------+----------+
				n, remoteAddr, err := localPC.ReadFrom(buf)
				if err != nil {
					log.Println(err)
					continue
				}

				tgtAddr := socks5.SplitAddr(buf[:n])
				if tgtAddr == nil {
					log.Printf("failed to split target address from packet: %q", buf[:n])
					continue
				}

				tgtUDPAddr, err := net.ResolveUDPAddr("udp", tgtAddr.String())
				if err != nil {
					log.Println(err)
					continue
				}

				payload := buf[len(tgtAddr):n]

				remotePC, err := net.ListenPacket("udp", "")
				if err != nil {
					log.Println(err)
					continue
				}

				socks5Req := socks.ParseUDPRequest(append([]byte{0, 0, 0}, buf[:n]...))
				targetHost, targetPort := socks.HostPort(socks5Req)
				log.Printf("proxy %s <-> %s <-> %s", remoteAddr, host, net.JoinHostPort(targetHost, targetPort))
				go func() {
					defer remotePC.Close()
					remoteBuf := make([]byte, MAXUDPPACKETSIZE)
					for {
						// The reassembly timer MUST be no less than 5 seconds?
						remotePC.SetReadDeadline(time.Now().Add(time.Second * 5))
						m, raddr, err := remotePC.ReadFrom(remoteBuf)
						if err != nil {
							if !util.IsEOF(err) && !util.IsTimeout(err) {
								log.Println(err)
							}
							break
						}

						srcAddr := socks5.ParseAddr(raddr.String())
						copy(remoteBuf[len(srcAddr):], remoteBuf[:m])
						copy(remoteBuf, srcAddr)
						_, err = localPC.WriteTo(remoteBuf[:len(srcAddr)+m], remoteAddr)
						if err != nil {
							log.Println(err)
							break
						}
					}
				}()
				_, err = remotePC.WriteTo(payload, tgtUDPAddr)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}(i, ss)
	}

	wg.Wait()
}
