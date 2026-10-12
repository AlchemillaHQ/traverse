package turn

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	turnserver "github.com/pion/turn/v4"
)

func newTURNFixture(t *testing.T, protocol string) string {
	t.Helper()
	const realm = "traverse-test"
	config := turnserver.ServerConfig{
		Realm: realm,
		AuthHandler: func(username, requestedRealm string, _ net.Addr) ([]byte, bool) {
			if username != "user" || requestedRealm != realm {
				return nil, false
			}
			return turnserver.GenerateAuthKey(username, realm, "password"), true
		},
	}
	generator := &turnserver.RelayAddressGeneratorStatic{RelayAddress: net.ParseIP("127.0.0.1"), Address: "127.0.0.1"}
	var address string
	if protocol == "tcp" {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { listener.Close() })
		address = listener.Addr().String()
		// No UDP control socket: a mislabeled UDP probe cannot pass this test.
		config.ListenerConfigs = []turnserver.ListenerConfig{{Listener: listener, RelayAddressGenerator: generator}}
	} else {
		conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		address = conn.LocalAddr().String()
		config.PacketConnConfigs = []turnserver.PacketConnConfig{{PacketConn: conn, RelayAddressGenerator: generator}}
	}
	server, err := turnserver.NewServer(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	return address
}

func TestAuthenticatedAllocation(t *testing.T) {
	for _, protocol := range []string{"udp", "tcp"} {
		t.Run(protocol, func(t *testing.T) {
			address := newTURNFixture(t, protocol)
			result := NewClient(address, "user", "password", protocol, time.Second, "127.0.0.1").Test(context.Background())
			if !result.Success || result.Error != "" || result.RelayedIP != "127.0.0.1" || result.RelayedPort == 0 {
				t.Fatalf("allocation failed: %+v", result)
			}
			if result.Protocol != protocol || result.LocalIP != "127.0.0.1" || result.Lifetime != "unknown" {
				t.Fatalf("incorrect allocation metadata: %+v", result)
			}
		})
	}
}

func TestRelayedConnectivity(t *testing.T) {
	address := newTURNFixture(t, "udp")
	result := NewClient(address, "user", "password", "udp", time.Second, "").TestConnectivity(context.Background())
	if !result.Success || result.Error != "" {
		t.Fatalf("relayed binding request failed: %+v", result)
	}
}

func TestBadCredentials(t *testing.T) {
	for _, protocol := range []string{"udp", "tcp"} {
		t.Run(protocol, func(t *testing.T) {
			address := newTURNFixture(t, protocol)
			result := NewClient(address, "user", "wrong-password", protocol, time.Second, "").Test(context.Background())
			if result.Success || result.Error == "" {
				t.Fatalf("bad credentials were accepted: %+v", result)
			}
		})
	}
}

func TestCancellationDuringAllocation(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan *Result, 1)
	go func() {
		results <- NewClient(conn.LocalAddr().String(), "user", "password", "udp", 5*time.Second, "").Test(ctx)
	}()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadFrom(make([]byte, 1500)); err != nil {
		t.Fatalf("probe never sent an allocation request: %v", err)
	}
	cancel()
	select {
	case result := <-results:
		if result.Success || !strings.Contains(result.Error, "context canceled") {
			t.Fatalf("unexpected cancellation result: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt the allocation transaction")
	}
}

func TestAllocationTimeout(t *testing.T) {
	for _, protocol := range []string{"udp", "tcp"} {
		t.Run(protocol, func(t *testing.T) {
			var address string
			if protocol == "tcp" {
				listener, err := net.Listen("tcp4", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer listener.Close()
				address = listener.Addr().String()
				done := make(chan struct{})
				go func() {
					defer close(done)
					conn, err := listener.Accept()
					if err == nil {
						defer conn.Close()
						buf := make([]byte, 1500)
						for {
							if _, err := conn.Read(buf); err != nil {
								return
							}
						}
					}
				}()
				defer func() { listener.Close(); <-done }()
			} else {
				conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				address = conn.LocalAddr().String()
			}
			start := time.Now()
			result := NewClient(address, "user", "password", protocol, 50*time.Millisecond, "").Test(context.Background())
			if result.Success || result.Error == "" || time.Since(start) > time.Second {
				t.Fatalf("allocation did not honor its timeout: %+v", result)
			}
		})
	}
}

func TestCanceledBeforeResolution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	result := NewClient("unresolved.example:3478", "user", "password", "udp", 5*time.Second, "").Test(ctx)
	if result.Success || !strings.Contains(result.Error, "context canceled") || time.Since(start) > time.Second {
		t.Fatalf("canceled probe was not stopped before resolution: %+v", result)
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, tt := range []struct{ username, password, protocol, localIP string }{
		{"", "password", "udp", ""},
		{"user", "", "udp", ""},
		{"user", "password", "invalid", ""},
		{"user", "password", "udp", "not-an-ip"},
	} {
		result := NewClient("127.0.0.1:3478", tt.username, tt.password, tt.protocol, time.Second, tt.localIP).Test(context.Background())
		if result.Success || result.Error == "" {
			t.Fatalf("invalid options were accepted: %+v", result)
		}
	}
}
