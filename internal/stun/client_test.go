package stun

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name		string
		serverAddr	string
		timeout		time.Duration
		localIP		string
		wantAddr	string
	}{
		{
			name:		"default timeout",
			serverAddr:	"stun.l.google.com:19302",
			timeout:	0,
			localIP:	"",
			wantAddr:	"stun.l.google.com:19302",
		},
		{
			name:		"custom timeout",
			serverAddr:	"stun.example.com:3478",
			timeout:	10 * time.Second,
			localIP:	"",
			wantAddr:	"stun.example.com:3478",
		},
		{
			name:		"with local IP",
			serverAddr:	"stun.l.google.com:19302",
			timeout:	3 * time.Second,
			localIP:	"192.168.1.1",
			wantAddr:	"stun.l.google.com:19302",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.serverAddr, tt.timeout, tt.localIP)
			if c.serverAddr != tt.wantAddr {
				t.Errorf("serverAddr = %q, want %q", c.serverAddr, tt.wantAddr)
			}
			if tt.timeout == 0 && c.timeout != 3*time.Second {
				t.Errorf("default timeout = %v, want 3s", c.timeout)
			}
			if tt.timeout > 0 && c.timeout != tt.timeout {
				t.Errorf("timeout = %v, want %v", c.timeout, tt.timeout)
			}
			if c.localIP != tt.localIP {
				t.Errorf("localIP = %q, want %q", c.localIP, tt.localIP)
			}
		})
	}
}

func TestResultStruct(t *testing.T) {
	r := &Result{
		Success:	true,
		Server:		"stun.example.com:3478",
		Duration:	"150ms",
		PublicIP:	"203.0.113.1",
		PublicPort:	54321,
		NATType:	"open_internet",
		LocalIP:	"10.0.0.1",
		Timestamp:	time.Now(),
	}

	if !r.Success {
		t.Error("expected Success to be true")
	}
	if r.Server != "stun.example.com:3478" {
		t.Errorf("Server = %q, want %q", r.Server, "stun.example.com:3478")
	}
	if r.PublicPort != 54321 {
		t.Errorf("PublicPort = %d, want 54321", r.PublicPort)
	}
}

func TestNATTypeConstants(t *testing.T) {
	types := []NATType{
		NATTypeUnknown,
		NATTypeOpenInternet,
		NATTypeFullCone,
		NATTypeRestrictedCone,
		NATTypePortRestrictedCone,
		NATTypeSymmetric,
		NATTypeSymmetricUDPFirewall,
	}

	seen := make(map[string]bool)
	for _, nt := range types {
		if string(nt) == "" {
			t.Error("NAT type should not be empty")
		}
		if seen[string(nt)] {
			t.Errorf("duplicate NAT type: %s", nt)
		}
		seen[string(nt)] = true
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip	string
		private	bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
		{"172.15.0.1", false},
		{"172.32.0.1", false},
		{"192.169.0.1", false},
		{"9.9.9.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid test IP: %s", tt.ip)
			}
			result := isPrivateIP(ip)
			if result != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.private)
			}
		})
	}
}

func TestDetermineNATType(t *testing.T) {
	tests := []struct {
		name	string
		localIP	string
		pubIP	string
		pubPort	int
		want	NATType
	}{
		{
			name:		"open internet",
			localIP:	"8.8.8.8",
			pubIP:		"8.8.8.8",
			pubPort:	12345,
			want:		NATTypeOpenInternet,
		},
		{
			name:		"behind NAT (symmetric guess)",
			localIP:	"192.168.1.100",
			pubIP:		"203.0.113.50",
			pubPort:	54321,
			want:		NATTypeSymmetric,
		},
		{
			name:		"CGNAT private",
			localIP:	"100.64.50.10",
			pubIP:		"203.0.113.50",
			pubPort:	54321,
			want:		NATTypeSymmetric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localIP := net.ParseIP(tt.localIP)
			pubIP := net.ParseIP(tt.pubIP)
			if pubIP == nil {
				t.Fatalf("invalid public IP: %s", tt.pubIP)
			}
			got := determineNATType(localIP, pubIP)
			if got != tt.want {
				t.Errorf("determineNATType() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTest_InvalidServer(t *testing.T) {
	c := NewClient("invalid.server.name.invalid:9999", 1*time.Second, "")
	ctx := context.Background()
	result := c.Test(ctx)

	if result.Success {
		t.Error("expected failure for invalid server")
	}
	if result.Error == "" {
		t.Error("expected error message for invalid server")
	}
	if result.Server != "invalid.server.name.invalid:9999" {
		t.Errorf("Server = %q, want %q", result.Server, "invalid.server.name.invalid:9999")
	}
}

func TestTest_ContextCancellation(t *testing.T) {
	c := NewClient("8.8.8.8:9999", 5*time.Second, "")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	result := c.Test(ctx)
	if result.Success {
		t.Error("expected failure with canceled context (may still succeed if server responds)")
	}
}

func TestClient_NoTimeout(t *testing.T) {

	c := NewClient("stun.l.google.com:19302", 0, "")
	if c.timeout != 3*time.Second {
		t.Errorf("expected default timeout of 3s, got %v", c.timeout)
	}
}

func TestClient_ServerAddressFormat(t *testing.T) {

	formats := []struct {
		addr	string
		ok	bool
	}{
		{"stun.l.google.com:19302", true},
		{"[::1]:3478", true},
		{"192.168.1.1:3478", true},
		{"example.com", false},
	}

	for _, f := range formats {
		t.Run(f.addr, func(t *testing.T) {
			c := NewClient(f.addr, 1*time.Second, "")
			if c.serverAddr != f.addr {
				t.Errorf("serverAddr = %q, want %q", c.serverAddr, f.addr)
			}
		})
	}
}
