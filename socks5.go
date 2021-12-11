package proxy

import (
	"net/url"

	netproxy "golang.org/x/net/proxy"
)

func SOCKS5(u *url.URL, forward netproxy.Dialer) (proxy Proxy, err error) {
	if u.Scheme != "socks5" && u.Scheme != "socks5h" {
		err = ErrInvalidParams
		return
	}

	return fromURL(u, forward)
}
