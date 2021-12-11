package proxy

import (
	"net/http"
	"net/url"
)

var Direct Proxy

func init() {
	u, _ := url.Parse("direct://localhost")
	Direct = &simpleProxyT{
		u:      u,
		client: http.DefaultClient,
	}
}
