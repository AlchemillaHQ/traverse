package stun

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	pionstun "github.com/pion/stun/v3"
)

func newSTUNFixture(t *testing.T, reply func(*pionstun.Message, *net.UDPAddr) *pionstun.Message) string {
	t.Helper()
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		conn.Close()
		<-done
	})
	go func() {
		defer close(done)
		buf := make([]byte, 1500)
		for {
			n, peer, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			request := &pionstun.Message{Raw: buf[:n]}
			if err := request.Decode(); err != nil {
				t.Errorf("fixture request: %v", err)
				return
			}
			if response := reply(request, peer.(*net.UDPAddr)); response != nil {
				if _, err := conn.WriteTo(response.Raw, peer); err != nil {
					t.Errorf("fixture response: %v", err)
					return
				}
			}
		}
	}()
	return conn.LocalAddr().String()
}

func TestBindingSuccess(t *testing.T) {
	for _, publicIP := range []string{"127.0.0.1", "203.0.113.42"} {
		t.Run(publicIP, func(t *testing.T) {
			address := newSTUNFixture(t, func(request *pionstun.Message, peer *net.UDPAddr) *pionstun.Message {
				return pionstun.MustBuild(pionstun.NewTransactionIDSetter(request.TransactionID), pionstun.BindingSuccess,
					&pionstun.XORMappedAddress{IP: net.ParseIP(publicIP), Port: peer.Port})
			})
			result := NewClient(address, time.Second, "127.0.0.1").Test(context.Background())
			if !result.Success || result.Error != "" || result.PublicIP != publicIP || result.PublicPort == 0 {
				t.Fatalf("unexpected result: %+v", result)
			}
			if result.LocalIP != "127.0.0.1" || result.NATType != string(NATTypeUnknown) {
				t.Fatalf("incorrect local address or guessed NAT type: %+v", result)
			}
			if duration, err := time.ParseDuration(result.Duration); err != nil || duration <= 0 {
				t.Fatalf("invalid duration: %q", result.Duration)
			}
		})
	}
}

func TestRejectWrongTransaction(t *testing.T) {
	address := newSTUNFixture(t, func(request *pionstun.Message, peer *net.UDPAddr) *pionstun.Message {
		id := request.TransactionID
		id[0] ^= 0xff
		return pionstun.MustBuild(pionstun.NewTransactionIDSetter(id), pionstun.BindingSuccess,
			&pionstun.XORMappedAddress{IP: peer.IP, Port: peer.Port})
	})
	result := NewClient(address, time.Second, "").Test(context.Background())
	if result.Success || !strings.Contains(result.Error, "transaction ID") {
		t.Fatalf("unrelated response was not rejected: %+v", result)
	}
}

func TestCancellationDuringRead(t *testing.T) {
	received := make(chan struct{}, 1)
	address := newSTUNFixture(t, func(*pionstun.Message, *net.UDPAddr) *pionstun.Message {
		select {
		case received <- struct{}{}:
		default:
		}
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan *Result, 1)
	go func() { resultCh <- NewClient(address, 5*time.Second, "").Test(ctx) }()
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("probe never sent its request")
	}
	cancel()
	select {
	case result := <-resultCh:
		if result.Success || !strings.Contains(result.Error, "context canceled") {
			t.Fatalf("unexpected result: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt the blocked read")
	}
}

func TestProbeTimeout(t *testing.T) {
	address := newSTUNFixture(t, func(*pionstun.Message, *net.UDPAddr) *pionstun.Message { return nil })
	for _, parentDeadline := range []bool{false, true} {
		ctx := context.Background()
		timeout := 50 * time.Millisecond
		if parentDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
			timeout = 5 * time.Second
		}
		start := time.Now()
		result := NewClient(address, timeout, "").Test(ctx)
		if result.Success || result.Error == "" || time.Since(start) > time.Second {
			t.Fatalf("probe did not honor its deadline: %+v", result)
		}
	}
}

func TestRejectInvalidLocalIP(t *testing.T) {
	result := NewClient("127.0.0.1:3478", time.Second, "not-an-ip").Test(context.Background())
	if result.Success || !strings.Contains(result.Error, "invalid local IP") {
		t.Fatalf("invalid local IP was silently ignored: %+v", result)
	}
}
