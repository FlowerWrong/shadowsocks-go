package util

import (
	"net/url"
)

// ParseSSURL ...
func ParseSSURL(ss string) (host, method, password, localPort string, err error) {
	u, err := url.Parse(ss)
	if err != nil {
		return
	}

	host = u.Host

	if u.User != nil {
		method = u.User.Username()
		password, _ = u.User.Password()
	}

	if len(u.Query()) > 0 {
		localPort = u.Query().Get("local_port")
	}
	return
}
