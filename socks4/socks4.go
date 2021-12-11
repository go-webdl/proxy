package socks4

import (
	"context"
	"errors"
	"net"
	"strconv"
)

// An Addr represents a SOCKS-specific address.
type Addr struct {
	IP   net.IP
	Port int
}

func (a *Addr) Network() string { return "socks" }

func (a *Addr) String() string {
	if a == nil {
		return "<nil>"
	}
	port := strconv.Itoa(a.Port)
	if a.IP == nil {
		return net.JoinHostPort("", port)
	}
	return net.JoinHostPort(a.IP.String(), port)
}

// A Conn represents a forward proxy connection.
type Conn struct {
	net.Conn

	boundAddr net.Addr
}

// BoundAddr returns the address assigned by the proxy server for
// connecting to the command target address from the proxy server.
func (c *Conn) BoundAddr() net.Addr {
	if c == nil {
		return nil
	}
	return c.boundAddr
}

// A Dialer holds SOCKS-specific options.
type Dialer struct {
	proxyNetwork string // network between a proxy server and a client
	proxyAddress string // proxy server address

	// ProxyDial specifies the optional dial function for
	// establishing the transport connection.
	ProxyDial func(context.Context, string, string) (net.Conn, error)

	// User ID
	User string

	// SOCKS4a DNS support
	SupportDNS bool
}

// DialContext connects to the provided address on the provided
// network.
//
// The returned error value may be a net.OpError. When the Op field of
// net.OpError contains "socks", the Source field contains a proxy
// server address and the Addr field contains a command target
// address.
//
// See func Dial of the net package of standard library for a
// description of the network and address parameters.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}
	var err error
	var c net.Conn
	if d.ProxyDial != nil {
		c, err = d.ProxyDial(ctx, d.proxyNetwork, d.proxyAddress)
	} else {
		var dd net.Dialer
		c, err = dd.DialContext(ctx, d.proxyNetwork, d.proxyAddress)
	}
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	a, err := d.connect(ctx, c, address)
	if err != nil {
		c.Close()
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	return &Conn{Conn: c, boundAddr: a}, nil
}

// DialWithConn initiates a connection from SOCKS server to the target
// network and address using the connection c that is already
// connected to the SOCKS server.
//
// It returns the connection's local address assigned by the SOCKS
// server.
func (d *Dialer) DialWithConn(ctx context.Context, c net.Conn, network, address string) (net.Addr, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}
	a, err := d.connect(ctx, c, address)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	return a, nil
}

// Dial connects to the provided address on the provided network.
//
// Unlike DialContext, it returns a raw transport connection instead
// of a forward proxy connection.
//
// Deprecated: Use DialContext or DialWithConn instead.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	var err error
	var c net.Conn
	if d.ProxyDial != nil {
		c, err = d.ProxyDial(context.Background(), d.proxyNetwork, d.proxyAddress)
	} else {
		c, err = net.Dial(d.proxyNetwork, d.proxyAddress)
	}
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: "connect", Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if _, err := d.DialWithConn(context.Background(), c, network, address); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (d *Dialer) validateTarget(network, address string) error {
	switch network {
	case "tcp", "tcp6", "tcp4":
	default:
		return errors.New("network not implemented")
	}
	return nil
}

func (d *Dialer) pathAddrs(address string) (proxy, dst net.Addr, err error) {
	for i, s := range []string{d.proxyAddress, address} {
		host, port, err := splitHostPort(s)
		if err != nil {
			return nil, nil, err
		}
		a := &Addr{Port: port}
		a.IP = net.ParseIP(host)
		if i == 0 {
			proxy = a
		} else {
			dst = a
		}
	}
	return
}

// NewDialer returns a new Dialer that dials through the provided
// proxy server's network and address.
func NewDialer(network, address string) *Dialer {
	return &Dialer{proxyNetwork: network, proxyAddress: address}
}
