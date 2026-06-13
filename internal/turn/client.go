package turn

import (
	"context"
	"fmt"
	"net"
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
		protocol:   protocol,
		timeout:    timeout,
		localIP:    localIP,
	}
}

func (c *Client) Test(ctx context.Context) *Result {
	result := &Result{
		Server:    c.serverAddr,
		Protocol:  c.protocol,
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

	localAddrVal := conn.LocalAddr().(*net.UDPAddr)
	result.LocalIP = localAddrVal.IP.String()

	cfg := &turnclient.ClientConfig{
		STUNServerAddr: c.serverAddr,
		TURNServerAddr: c.serverAddr,
		Conn:           conn,
		Username:       c.username,
		Password:       c.password,
		Realm:          "",
	}

	tc, err := turnclient.NewClient(cfg)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create TURN client: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer tc.Close()

	relayConn, err := tc.Allocate()
	if err != nil {
		result.Error = fmt.Sprintf("failed to allocate relay address: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer relayConn.Close()

	relayAddr := relayConn.LocalAddr().(*net.UDPAddr)
	result.RelayedIP = relayAddr.IP.String()
	result.RelayedPort = relayAddr.Port

	result.Lifetime = "600s (default)"

	result.Success = true
	result.Duration = time.Since(start).String()

	return result
}

func (c *Client) TestConnectivity(ctx context.Context) *Result {
	result := &Result{
		Server:    c.serverAddr,
		Protocol:  c.protocol,
		Timestamp: time.Now(),
	}

	start := time.Now()

	serverAddr, err := net.ResolveUDPAddr("udp", c.serverAddr)
	if err != nil {
		result.Error = fmt.Sprintf("resolve: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	var localAddr *net.UDPAddr
	if c.localIP != "" {
		localAddr = &net.UDPAddr{IP: net.ParseIP(c.localIP)}
	}

	conn, err := net.DialUDP("udp", localAddr, serverAddr)
	if err != nil {
		result.Error = fmt.Sprintf("dial: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		result.Error = fmt.Sprintf("deadline: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	cfg := &turnclient.ClientConfig{
		STUNServerAddr: c.serverAddr,
		TURNServerAddr: c.serverAddr,
		Conn:           conn,
		Username:       c.username,
		Password:       c.password,
		Realm:          "",
	}

	tc, err := turnclient.NewClient(cfg)
	if err != nil {
		result.Error = fmt.Sprintf("create client: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer tc.Close()

	relayConn, err := tc.Allocate()
	if err != nil {
		result.Error = fmt.Sprintf("allocate: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}
	defer relayConn.Close()

	relayAddr := relayConn.LocalAddr().(*net.UDPAddr)
	result.RelayedIP = relayAddr.IP.String()
	result.RelayedPort = relayAddr.Port
	result.LocalIP = conn.LocalAddr().(*net.UDPAddr).IP.String()

	testMsg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if err := tc.CreatePermission(&net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53}); err != nil {

		_ = err
	}

	if _, err := relayConn.WriteTo(testMsg.Raw, serverAddr); err != nil {
		result.Error = fmt.Sprintf("relay send: %v", err)
		result.Duration = time.Since(start).String()
		return result
	}

	result.Success = true
	result.Duration = time.Since(start).String()

	return result
}
