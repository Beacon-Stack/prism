package safedialer

import (
	"context"
	"net"
	"testing"
)

func TestIsBlocked_Strict(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// Loopback
		{"ipv4 loopback", "127.0.0.1", true},
		{"ipv4 loopback high", "127.255.255.255", true},
		{"ipv6 loopback", "::1", true},

		// RFC-1918
		{"10.x", "10.0.0.1", true},
		{"10.x high", "10.255.255.255", true},
		{"172.16.x", "172.16.0.1", true},
		{"172.31.x", "172.31.255.255", true},
		{"192.168.x", "192.168.1.1", true},
		{"192.168.x high", "192.168.255.255", true},

		// Link-local / cloud metadata
		{"link-local", "169.254.0.1", true},
		{"cloud metadata", "169.254.169.254", true},
		{"ipv6 link-local", "fe80::1", true},

		// CGNAT / Tailscale
		{"cgnat low", "100.64.0.1", true},
		{"cgnat high", "100.127.255.255", true},

		// Unspecified
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v4 range", "0.255.255.255", true},

		// IPv6 unique-local
		{"ipv6 ula", "fd00::1", true},

		// Public IPs - should NOT be blocked
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public 93.184.216.34", "93.184.216.34", false},
		{"public ipv6", "2001:db8::1", false},

		// Edge cases near RFC-1918 boundaries
		{"just outside 172.16", "172.15.255.255", false},
		{"just outside 172.32", "172.32.0.0", false},
		{"just outside 192.168", "192.167.255.255", false},
		{"just outside 10", "11.0.0.0", false},
		{"just outside cgnat", "100.128.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid test IP: %s", tt.ip)
			}
			got := isBlocked(ip)
			if got != tt.blocked {
				t.Errorf("isBlocked(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestIsLANBlocked(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// Cloud metadata - blocked even in LAN mode
		{"cloud metadata", "169.254.169.254", true},
		{"link-local", "169.254.0.1", true},
		{"ipv6 link-local", "fe80::1", true},
		{"cgnat", "100.64.0.1", true},

		// RFC-1918 - allowed in LAN mode
		{"10.x allowed", "10.0.0.1", false},
		{"172.16.x allowed", "172.16.0.1", false},
		{"192.168.x allowed", "192.168.1.1", false},

		// Loopback - allowed in LAN mode
		{"loopback allowed", "127.0.0.1", false},
		{"ipv6 loopback allowed", "::1", false},

		// Public - allowed
		{"public", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid test IP: %s", tt.ip)
			}
			got := isLANBlocked(ip)
			if got != tt.blocked {
				t.Errorf("isLANBlocked(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestTransport_ReturnsNonNil(t *testing.T) {
	tr := Transport()
	if tr == nil {
		t.Fatal("Transport() returned nil")
	}
	if tr.DialContext == nil {
		t.Fatal("Transport().DialContext is nil")
	}
}

func TestLANTransport_ReturnsNonNil(t *testing.T) {
	tr := LANTransport()
	if tr == nil {
		t.Fatal("LANTransport() returned nil")
	}
	if tr.DialContext == nil {
		t.Fatal("LANTransport().DialContext is nil")
	}
}

func TestDialContext_BlocksLoopback(t *testing.T) {
	_, err := DialContext(context.Background(), "tcp", "127.0.0.1:80")
	if err == nil {
		t.Fatal("DialContext to 127.0.0.1 should be blocked")
	}
}

func TestDialContext_BlocksPrivate(t *testing.T) {
	for _, addr := range []string{
		"10.0.0.1:80",
		"172.16.0.1:80",
		"192.168.1.1:80",
	} {
		_, err := DialContext(context.Background(), "tcp", addr)
		if err == nil {
			t.Errorf("DialContext to %s should be blocked", addr)
		}
	}
}

func TestDialContext_BlocksMetadata(t *testing.T) {
	_, err := DialContext(context.Background(), "tcp", "169.254.169.254:80")
	if err == nil {
		t.Fatal("DialContext to cloud metadata should be blocked")
	}
}

func TestDialContext_InvalidAddr(t *testing.T) {
	_, err := DialContext(context.Background(), "tcp", "not-a-valid-addr")
	if err == nil {
		t.Fatal("DialContext with no port should return error")
	}
}
