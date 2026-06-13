package turn

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name		string
		serverAddr	string
		username	string
		password	string
		protocol	string
		timeout		time.Duration
		localIP		string
		wantProto	string
	}{
		{
			name:		"default protocol udp",
			serverAddr:	"turn.example.com:3478",
			username:	"user",
			password:	"pass",
			protocol:	"",
			timeout:	0,
			localIP:	"",
			wantProto:	"udp",
		},
		{
			name:		"custom protocol tcp",
			serverAddr:	"turn.example.com:3478",
			username:	"user",
			password:	"pass",
			protocol:	"tcp",
			timeout:	10 * time.Second,
			localIP:	"",
			wantProto:	"tcp",
		},
		{
			name:		"with local IP",
			serverAddr:	"turn.example.com:3478",
			username:	"user",
			password:	"pass",
			protocol:	"udp",
			timeout:	5 * time.Second,
			localIP:	"10.0.0.1",
			wantProto:	"udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.serverAddr, tt.username, tt.password, tt.protocol, tt.timeout, tt.localIP)
			if c.serverAddr != tt.serverAddr {
				t.Errorf("serverAddr = %q, want %q", c.serverAddr, tt.serverAddr)
			}
			if c.username != tt.username {
				t.Errorf("username = %q, want %q", c.username, tt.username)
			}
			if c.password != tt.password {
				t.Errorf("password = %q, want %q", c.password, tt.password)
			}
			if c.protocol != tt.wantProto {
				t.Errorf("protocol = %q, want %q", c.protocol, tt.wantProto)
			}
			if tt.timeout == 0 && c.timeout != 5*time.Second {
				t.Errorf("default timeout = %v, want 5s", c.timeout)
			}
			if tt.timeout > 0 && c.timeout != tt.timeout {
				t.Errorf("timeout = %v, want %v", c.timeout, tt.timeout)
			}
		})
	}
}

func TestResultStruct(t *testing.T) {
	r := &Result{
		Success:	true,
		Server:		"turn.example.com:3478",
		Duration:	"250ms",
		RelayedIP:	"203.0.113.10",
		RelayedPort:	49152,
		Lifetime:	"600s",
		LocalIP:	"10.0.0.1",
		Protocol:	"udp",
		Timestamp:	time.Now(),
	}

	if !r.Success {
		t.Error("expected Success to be true")
	}
	if r.Server != "turn.example.com:3478" {
		t.Errorf("Server = %q, want %q", r.Server, "turn.example.com:3478")
	}
	if r.RelayedPort != 49152 {
		t.Errorf("RelayedPort = %d, want 49152", r.RelayedPort)
	}
	if r.Protocol != "udp" {
		t.Errorf("Protocol = %q, want udp", r.Protocol)
	}
}

func TestTest_InvalidServer(t *testing.T) {
	c := NewClient("invalid.turn.server.invalid:9999", "user", "pass", "udp", 1*time.Second, "")
	ctx := context.Background()
	result := c.Test(ctx)

	if result.Success {
		t.Error("expected failure for invalid TURN server")
	}
	if result.Error == "" {
		t.Error("expected error message for invalid TURN server")
	}
}

func TestTest_MissingCredentials(t *testing.T) {

	c := NewClient("turn.example.com:3478", "", "", "udp", 5*time.Second, "")
	if c.username != "" {
		t.Error("expected empty username")
	}
	if c.password != "" {
		t.Error("expected empty password")
	}
}

func TestTestConnectivity_InvalidServer(t *testing.T) {
	c := NewClient("invalid.server.test:1", "user", "pass", "udp", 1*time.Second, "")
	ctx := context.Background()
	result := c.TestConnectivity(ctx)

	if result.Success {
		t.Error("expected failure for invalid server")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
}

func TestClient_ProtocolNormalization(t *testing.T) {

	c := NewClient("turn.example.com:3478", "u", "p", "", 0, "")
	if c.protocol != "udp" {
		t.Errorf("expected protocol 'udp' for empty input, got %q", c.protocol)
	}

	c2 := NewClient("turn.example.com:3478", "u", "p", "tcp", 0, "")
	if c2.protocol != "tcp" {
		t.Errorf("expected protocol 'tcp', got %q", c2.protocol)
	}
}

func TestClient_IPv6Address(t *testing.T) {
	c := NewClient("[::1]:3478", "user", "pass", "udp", 5*time.Second, "")
	if c.serverAddr != "[::1]:3478" {
		t.Errorf("serverAddr = %q, want [::1]:3478", c.serverAddr)
	}
}
