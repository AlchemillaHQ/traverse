package stun

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pion/stun/v3"
)

type Result struct {
	Success    bool      `json:"success" xml:"success"`
	Server     string    `json:"server" xml:"server"`
	Duration   string    `json:"duration" xml:"duration"`
	PublicIP   string    `json:"public_ip,omitempty" xml:"PublicIP,omitempty"`
	PublicPort int       `json:"public_port,omitempty" xml:"PublicPort,omitempty"`
	NATType    string    `json:"nat_type,omitempty" xml:"NATType,omitempty"`
	LocalIP    string    `json:"local_ip,omitempty" xml:"LocalIP,omitempty"`
	Error      string    `json:"error,omitempty" xml:"Error,omitempty"`
	Timestamp  time.Time `json:"timestamp" xml:"timestamp"`
}

type NATType string

const (
	NATTypeUnknown              NATType = "unknown"
	NATTypeOpenInternet         NATType = "open_internet"
	NATTypeFullCone             NATType = "full_cone"
	NATTypeRestrictedCone       NATType = "restricted_cone"
	NATTypePortRestrictedCone   NATType = "port_restricted_cone"
	NATTypeSymmetric            NATType = "symmetric"
	NATTypeSymmetricUDPFirewall NATType = "symmetric_udp_firewall"
)

type Client struct {
	serverAddr string
	timeout    time.Duration
	localIP    string
}

func NewClient(serverAddr string, timeout time.Duration, localIP string) *Client {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Client{
		serverAddr: serverAddr,
		timeout:    timeout,
		localIP:    localIP,
	}
}

func (c *Client) Test(ctx context.Context) *Result {
	result := &Result{
		Server:    c.serverAddr,
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

	dialer := &net.Dialer{}
	if c.localIP != "" {
		ip := net.ParseIP(c.localIP)
		if ip == nil {
			result.Error = fmt.Sprintf("invalid local IP: %q", c.localIP)
			return result
		}
		dialer.LocalAddr = &net.UDPAddr{IP: ip}
	}

	// DialContext also bounds DNS resolution, not just socket I/O.
	conn, err := dialer.DialContext(ctx, "udp", c.serverAddr)
	if err != nil {
		return fail("failed to dial UDP", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return fail("failed to set deadline", err)
	}

	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if _, err := conn.Write(msg.Raw); err != nil {
		return fail("failed to send binding request", err)
	}

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return fail("failed to read response", err)
	}

	res := &stun.Message{Raw: buf[:n]}
	if err := res.Decode(); err != nil {
		return fail("failed to decode response", err)
	}
	if res.TransactionID != msg.TransactionID {
		result.Error = "response transaction ID does not match the binding request"
		return result
	}

	if res.Type != stun.BindingSuccess {
		result.Error = fmt.Sprintf("unexpected response type: %s", res.Type)
		return result
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(res); err != nil {
		return fail("failed to get XOR-MAPPED-ADDRESS", err)
	}
	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.PublicIP = xorAddr.IP.String()
	result.PublicPort = xorAddr.Port

	localAddrVal := conn.LocalAddr().(*net.UDPAddr)
	result.LocalIP = localAddrVal.IP.String()

	result.NATType = string(determineNATType(localAddrVal.IP, xorAddr.IP))

	return result
}

func determineNATType(_, _ net.IP) NATType {
	// A single binding response cannot determine mapping or filtering behavior,
	// even when the local and mapped IP addresses are identical.
	return NATTypeUnknown
}
