// Code is copy from https://medium.com/@mlowicki/http-s-proxy-in-golang-in-less-than-100-lines-of-code-6a51c2f2c38c
// TODO https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers#hbh
package main

import (
	"crypto/tls"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"time"
)

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

// https proxy
func handleTunneling(w http.ResponseWriter, r *http.Request) {
	log.Println("proxy https")
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Any successful (2xx) response to a CONNECT request indicates that the
	// proxy has established a connection to the requested host and port,
	// and has switched to tunneling the current connection to that server
	// connection.
	// See https://www.ietf.org/rfc/rfc2817.txt
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	// 转变为普通tcp连接
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	// 对倒tcp flow
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

// http proxy
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	log.Println("proxy http")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable) // 503
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// go run cmd/http11_proxy.go
// curl -v -i -x http://127.0.0.1:8080 'https://baidu.com'
// curl -v -i -x http://127.0.0.1:8080 'http://baidu.com'
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				// https or websocket
				handleTunneling(w, r)
			} else {
				// http
				handleHTTP(w, r)
			}
		}),
		// disable HTTP2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Fatal(server.ListenAndServe())
}
