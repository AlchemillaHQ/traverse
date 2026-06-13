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

	serverAddr, err := net.ResolveUDPAddr("udp", c.serverAddr)
	if err != nil {
		result.Error = fmt.Sprintf("failed to resolve server address: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	var localAddr *net.UDPAddr
	if c.localIP != "" {
		localAddr = &net.UDPAddr{IP: net.ParseIP(c.localIP)}
	}

	conn, err := net.DialUDP("udp", localAddr, serverAddr)
	if err != nil {
		result.Error = fmt.Sprintf("failed to dial UDP: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		result.Error = fmt.Sprintf("failed to set deadline: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if _, err := conn.Write(msg.Raw); err != nil {
		result.Error = fmt.Sprintf("failed to send binding request: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	result.Duration = time.Since(start).String()

	res := &stun.Message{Raw: buf[:n]}
	if err := res.Decode(); err != nil {
		result.Error = fmt.Sprintf("failed to decode response: %v", err)
		return result
	}

	if res.Type != stun.BindingSuccess {
		result.Error = fmt.Sprintf("unexpected response type: %s", res.Type)
		return result
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(res); err != nil {
		result.Error = fmt.Sprintf("failed to get XOR-MAPPED-ADDRESS: %v", err)
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

func determineNATType(localIP, publicIP net.IP) NATType {

	if localIP.Equal(publicIP) {
		return NATTypeOpenInternet
	}

	if isPrivateIP(localIP) {

		return NATTypeSymmetric
	}

	return NATTypeUnknown
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {

		if ip4[0] == 10 {
			return true
		}

		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}

		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}

		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}
