// Package safedialer provides an http.Transport that blocks requests to
// private, loopback, and link-local network addresses, preventing SSRF attacks.
//
// All user-supplied URLs (Radarr import, indexer test/search, download client
// test/add, webhook notifications, torrent file fetches) must use this transport
// rather than a plain http.Client, so that an authenticated user cannot instruct
// the server to reach internal services.
package safedialer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// blockedCIDRs is the list of address ranges that must never be reachable via
// user-supplied URLs. Covers: loopback, private, link-local, and unspecified.
var blockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC-1918 private
		"172.16.0.0/12",  // RFC-1918 private
		"192.168.0.0/16", // RFC-1918 private
		"169.254.0.0/16", // IPv4 link-local (AWS/GCP metadata: 169.254.169.254)
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique-local
		"0.0.0.0/8",      // "this" network
		"100.64.0.0/10",  // CGNAT / Tailscale
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			panic("safedialer: invalid CIDR " + c + ": " + err.Error())
		}
		nets = append(nets, ipnet)
	}
	return nets
}()

// isBlocked returns true if ip falls within any of the blocked ranges.
func isBlocked(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// dialContext is a net.Dialer.DialContext replacement that resolves the host
// and rejects any resulting IP that falls within a blocked range.
func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("safedialer: parsing address %q: %w", addr, err)
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("safedialer: resolving %q: %w", host, err)
	}

	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			return nil, fmt.Errorf("safedialer: could not parse resolved IP %q", a)
		}
		if isBlocked(ip) {
			return nil, fmt.Errorf("safedialer: connection to %s (%s) is not allowed", host, ip)
		}
	}

	// All resolved addresses are public — proceed with the standard dialer.
	d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(addrs[0], port))
}

// Transport returns an *http.Transport that blocks requests to private/internal
// network addresses. Use this in place of http.DefaultTransport for all HTTP
// clients that connect to user-supplied URLs.
func Transport() *http.Transport {
	return &http.Transport{
		DialContext:           dialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
	}
}
