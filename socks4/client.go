package socks4

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	socks_version = 0x04
	socks_connect = 0x01
	socks_bind    = 0x02

	access_granted         = 0x5a
	access_rejected        = 0x5b
	access_identd_required = 0x5c
	access_identd_failed   = 0x5d
)

var (
	noDeadline   = time.Time{}
	aLongTimeAgo = time.Unix(1, 0)
)

func (d *Dialer) connect(ctx context.Context, c net.Conn, address string) (addr *Addr, ctxErr error) {
	host, port, err := splitHostPort(address)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok && !deadline.IsZero() {
		c.SetDeadline(deadline)
		defer c.SetDeadline(noDeadline)
	}
	if ctx != context.Background() {
		errCh := make(chan error, 1)
		done := make(chan struct{})
		defer func() {
			close(done)
			if ctxErr == nil {
				ctxErr = <-errCh
			}
		}()
		go func() {
			select {
			case <-ctx.Done():
				c.SetDeadline(aLongTimeAgo)
				errCh <- ctx.Err()
			case <-done:
				errCh <- nil
			}
		}()
	}

	ip4 := net.ParseIP(address)
	addrIsIP := ip4 != nil

	var b []byte
	if d.SupportDNS && !addrIsIP {
		b = make([]byte, 0, 10+len(d.User)+len(address))
	} else {
		b = make([]byte, 0, 9+len(d.User))
	}

	if !d.SupportDNS && !addrIsIP {
		var ip *net.IPAddr
		ip, ctxErr = net.ResolveIPAddr("ip4", host)
		if err != nil {
			ctxErr = fmt.Errorf("unable to find IP address of host: %w", err)
			return
		}
		ip4 = ip.IP.To4()
	}

	b = append(b, socks_version, socks_connect, byte(port>>8), byte(port))
	if ip4 != nil {
		b = append(b, ip4...)
	} else {
		b = append(b, 0, 0, 0, 1)
	}
	b = append(b, d.User...)
	b = append(b, 0)
	if d.SupportDNS && !addrIsIP {
		b = append(b, address...)
		b = append(b, 0)
	}

	if _, ctxErr = c.Write(b); ctxErr != nil {
		return
	}

	if _, ctxErr = io.ReadFull(c, b[:8]); ctxErr != nil {
		return
	}

	switch b[1] {
	case access_granted:
	case access_identd_required, access_identd_failed:
		ctxErr = fmt.Errorf("socks4 server require valid identd: %d", b[1])
		return
	default:
		ctxErr = fmt.Errorf("connection rejected: %d", b[1])
		return
	}

	addr = &Addr{IP: make(net.IP, net.IPv4len)}
	copy(addr.IP, b[4:8])
	addr.Port = int(b[2])<<8 | int(b[3])
	return
}

func splitHostPort(address string) (string, int, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	portnum, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	if 1 > portnum || portnum > 0xffff {
		return "", 0, errors.New("port number out of range " + port)
	}
	return host, portnum, nil
}
