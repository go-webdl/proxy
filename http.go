package proxy

import (
	"net/http"
	"net/url"

	netproxy "golang.org/x/net/proxy"
)

func HTTP(u *url.URL, forward netproxy.Dialer) (proxy Proxy, err error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		err = ErrInvalidParams
		return
	}

	proxy = &simpleProxyT{
		u: u,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
		},
	}

	return
}
