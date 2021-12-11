package proxy

import (
	"bufio"
	"os"
)

func ParseProxyURIString(proxies *[]Proxy, proxyURI string) (err error) {
	var p Proxy
	if proxyURI != "" {
		if p, err = FromURLString(proxyURI, nil); err != nil {
			return
		}
		*proxies = append(*proxies, p)
	}
	return
}

func ParseProxyFile(proxies *[]Proxy, proxyFile string) (err error) {
	file, err := os.Open(proxyFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err = ParseProxyURIString(proxies, scanner.Text()); err != nil {
			return
		}
	}

	err = scanner.Err()
	return
}

func ParseProxies(proxies *[]Proxy, proxyURI string, proxyListFiles []string) (err error) {
	if err = ParseProxyURIString(proxies, proxyURI); err != nil {
		return
	}

	for _, proxyFile := range proxyListFiles {
		if err = ParseProxyFile(proxies, proxyFile); err != nil {
			return
		}
	}

	if len(*proxies) == 0 {
		*proxies = append(*proxies, Direct)
	}

	return
}
