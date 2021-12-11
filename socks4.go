package proxy

import (
	"context"
	"net"
	"net/url"

	"github.com/go-webdl/proxy/socks4"

	netproxy "golang.org/x/net/proxy"
)

func SOCKS4(u *url.URL, forward netproxy.Dialer) (proxy Proxy, err error) {
	if u.Scheme != "socks4" && u.Scheme != "socks4a" {
		err = ErrInvalidParams
		return
	}

	return fromURL(u, forward)
}

func SOCKS4Dialer(u *url.URL, forward netproxy.Dialer) (dialer netproxy.Dialer, err error) {
	var user string
	if u.User != nil {
		user = u.User.Username()
	}
	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Hostname(), "1080")
	}
	d := socks4.NewDialer("tcp", u.Host)
	d.User = user
	d.SupportDNS = u.Scheme == "socks4a"
	if forward != nil {
		if f, ok := forward.(netproxy.ContextDialer); ok {
			d.ProxyDial = func(ctx context.Context, network string, address string) (net.Conn, error) {
				return f.DialContext(ctx, network, address)
			}
		} else {
			d.ProxyDial = func(ctx context.Context, network string, address string) (net.Conn, error) {
				return dialContext(ctx, forward, network, address)
			}
		}
	}
	return d, nil
}

func init() {
	netproxy.RegisterDialerType("socks4", SOCKS4Dialer)
	netproxy.RegisterDialerType("socks4a", SOCKS4Dialer)
}
