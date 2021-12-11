package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"

	netproxy "golang.org/x/net/proxy"
)

var (
	ErrUnknownScheme = errors.New("unknown proxy URI scheme")
	ErrInvalidParams = errors.New("invalid parameters")
)

type Proxy interface {
	URL() *url.URL
	Client() *http.Client
}

type simpleProxyT struct {
	u       *url.URL
	client  *http.Client
	forward netproxy.Dialer
}

var _ Proxy = (*simpleProxyT)(nil)

func (p *simpleProxyT) URL() *url.URL {
	return p.u
}

func (p *simpleProxyT) Client() *http.Client {
	return p.client
}

func FromURLString(u string, forward netproxy.Dialer) (proxy Proxy, err error) {
	var pu *url.URL
	if pu, err = url.Parse(u); err != nil {
		return
	}
	return FromURL(pu, forward)
}

func FromURL(u *url.URL, forward netproxy.Dialer) (proxy Proxy, err error) {
	switch u.Scheme {
	case "direct":
		proxy = Direct
	case "http", "https":
		proxy, err = HTTP(u, forward)
	case "socks4", "socks4a":
		proxy, err = SOCKS4(u, forward)
	case "socks5", "socks5h":
		proxy, err = SOCKS5(u, forward)
	default:
		err = ErrUnknownScheme
	}
	return
}

func fromURL(u *url.URL, forward netproxy.Dialer) (proxy Proxy, err error) {
	var dialer netproxy.Dialer
	if dialer, err = netproxy.FromURL(u, forward); err != nil {
		return
	}
	proxy = &simpleProxyT{
		u: u,
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
					return dialer.Dial(network, addr)
				},
			},
		},
	}
	return
}

// WARNING: this can leak a goroutine for as long as the underlying Dialer implementation takes to timeout
// A Conn returned from a successful Dial after the context has been cancelled will be immediately closed.
func dialContext(ctx context.Context, d netproxy.Dialer, network, address string) (net.Conn, error) {
	var (
		conn net.Conn
		done = make(chan struct{}, 1)
		err  error
	)
	go func() {
		conn, err = d.Dial(network, address)
		close(done)
		if conn != nil && ctx.Err() != nil {
			conn.Close()
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-done:
	}
	return conn, err
}
