package turn

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/stun/v3"
	turnclient "github.com/pion/turn/v4"
)

type Result struct {
	Success     bool      `json:"success" xml:"success"`
	Server      string    `json:"server" xml:"server"`
	Duration    string    `json:"duration" xml:"duration"`
	RelayedIP   string    `json:"relayed_ip,omitempty" xml:"RelayedIP,omitempty"`
	RelayedPort int       `json:"relayed_port,omitempty" xml:"RelayedPort,omitempty"`
	Lifetime    string    `json:"lifetime,omitempty" xml:"Lifetime,omitempty"`
	LocalIP     string    `json:"local_ip,omitempty" xml:"LocalIP,omitempty"`
	Protocol    string    `json:"protocol" xml:"protocol"`
	Error       string    `json:"error,omitempty" xml:"Error,omitempty"`
	Timestamp   time.Time `json:"timestamp" xml:"timestamp"`
}

type Client struct {
	serverAddr string
	username   string
	password   string
	protocol   string
	timeout    time.Duration
	localIP    string
}

func NewClient(serverAddr, username, password, protocol string, timeout time.Duration, localIP string) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if protocol == "" {
		protocol = "udp"
	}
	return &Client{
		serverAddr: serverAddr,
		username:   username,
		password:   password,
		protocol:   strings.ToLower(protocol),
		timeout:    timeout,
		localIP:    localIP,
	}
}

func (c *Client) Test(ctx context.Context) *Result {
	return c.test(ctx, false)
}

// TestConnectivity also sends a binding request through the relay. The TURN
// endpoint must accept UDP STUN requests at the same address and port.
func (c *Client) TestConnectivity(ctx context.Context) *Result {
	return c.test(ctx, true)
}

// connect selects a real TCP transport or an unconnected UDP packet socket.
// Pion v4 resolves TURN endpoints as udp4, including TCP control connections.
func (c *Client) connect(ctx context.Context) (net.PacketConn, *net.UDPAddr, error) {
	dialer := &net.Dialer{}
	if c.localIP != "" {
		ip := net.ParseIP(c.localIP)
		if ip == nil || ip.To4() == nil {
			return nil, nil, fmt.Errorf("TURN requires a valid IPv4 local IP, got %q", c.localIP)
		}
		if c.protocol == "tcp" {
			dialer.LocalAddr = &net.TCPAddr{IP: ip}
		} else {
			dialer.LocalAddr = &net.UDPAddr{IP: ip}
		}
	}
	if c.protocol != "udp" && c.protocol != "tcp" {
		return nil, nil, fmt.Errorf("unsupported TURN protocol: %q", c.protocol)
	}
	conn, err := dialer.DialContext(ctx, c.protocol+"4", c.serverAddr)
	if err != nil {
		return nil, nil, err
	}
	if c.protocol == "tcp" {
		remote := conn.RemoteAddr().(*net.TCPAddr)
		return turnclient.NewSTUNConn(conn), &net.UDPAddr{IP: remote.IP, Port: remote.Port}, nil
	}
	// This temporary connected socket resolves the endpoint and selects a route
	// without sending a packet. Bind a separate socket to the selected local IP:
	// Pion needs WriteTo/ReadFrom, which cannot use a connected UDP socket.
	remote := conn.RemoteAddr().(*net.UDPAddr)
	local := conn.LocalAddr().(*net.UDPAddr)
	bind := (&net.UDPAddr{IP: local.IP}).String()
	conn.Close()
	var listener net.ListenConfig
	packet, err := listener.ListenPacket(ctx, "udp4", bind)
	if err != nil {
		return nil, nil, err
	}
	return packet, remote, nil
}

func (c *Client) test(ctx context.Context, connectivity bool) *Result {
	result := &Result{
		Server:    c.serverAddr,
		Protocol:  c.protocol,
		Timestamp: time.Now(),
	}

	start := time.Now()
	defer func() { result.Duration = time.Since(start).String() }()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result
	}
	fail := func(message string, err error) *Result {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		result.Error = fmt.Sprintf("%s: %v", message, err)
		return result
	}
	if c.username == "" || c.password == "" {
		result.Error = "TURN username and password are required"
		return result
	}
	conn, serverAddr, err := c.connect(ctx)
	if err != nil {
		return fail("failed to connect to TURN server", err)
	}
	defer conn.Close()

	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fail("failed to set deadline", err)
	}
	result.LocalIP, _, _ = net.SplitHostPort(conn.LocalAddr().String())

	cfg := &turnclient.ClientConfig{
		STUNServerAddr: serverAddr.String(),
		TURNServerAddr: serverAddr.String(),
		Conn:           conn,
		Username:       c.username,
		Password:       c.password,
		Realm:          "",
	}

	tc, err := turnclient.NewClient(cfg)
	if err != nil {
		return fail("failed to create TURN client", err)
	}
	defer tc.Close()
	// Closing the socket alone does not wake Pion's pending transactions.
	stop := context.AfterFunc(ctx, func() {
		conn.Close()
		tc.Close()
	})
	defer stop()
	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result
	}
	if err := tc.Listen(); err != nil {
		return fail("failed to listen for TURN responses", err)
	}

	relayConn, err := tc.Allocate()
	if err != nil {
		return fail("failed to allocate relay address", err)
	}
	defer relayConn.Close()

	relayAddr := relayConn.LocalAddr().(*net.UDPAddr)
	result.RelayedIP = relayAddr.IP.String()
	result.RelayedPort = relayAddr.Port

	// Pion does not expose the negotiated allocation lifetime.
	result.Lifetime = "unknown"

	if connectivity {
		if err := relayConn.SetDeadline(deadline); err != nil {
			return fail("failed to set relay deadline", err)
		}
		stopRelay := context.AfterFunc(ctx, func() { relayConn.SetReadDeadline(time.Now()) })
		defer stopRelay()
		if err := tc.CreatePermission(serverAddr); err != nil {
			return fail("failed to create relay permission", err)
		}
		msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
		if _, err := relayConn.WriteTo(msg.Raw, serverAddr); err != nil {
			return fail("failed to send relayed binding request", err)
		}
		buf := make([]byte, 1500)
		n, peer, err := relayConn.ReadFrom(buf)
		if err != nil {
			return fail("failed to read relayed response", err)
		}
		res := &stun.Message{Raw: buf[:n]}
		if err := res.Decode(); err != nil {
			return fail("failed to decode relayed response", err)
		}
		if peer.String() != serverAddr.String() || res.Type != stun.BindingSuccess || res.TransactionID != msg.TransactionID {
			result.Error = "relayed response does not match the binding request"
			return result
		}
	}
	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	return result
}
