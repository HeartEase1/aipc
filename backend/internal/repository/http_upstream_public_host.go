package repository

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

// Pin untrusted download destinations through DNS, proxy CONNECT and TLS while
// preserving the original Host header and certificate name.
type publicHostTransport struct {
	base   http.RoundTripper
	lookup func(context.Context, string) ([]net.IPAddr, error)
}

func (t *publicHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base, ok := t.base.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("public image downloads require a standard HTTP transport")
	}
	host := req.URL.Hostname()
	if urlvalidator.IsBlockedHost(host) || (req.URL.Scheme != "https" && req.URL.Scheme != "http") {
		return nil, fmt.Errorf("image download destination is not public")
	}
	lookup := t.lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	ips, err := lookup(req.Context(), host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("cannot resolve image download destination")
	}
	for _, ip := range ips {
		if ip.Zone != "" || !ip.IP.IsGlobalUnicast() || urlvalidator.IsBlockedHost(ip.IP.String()) {
			return nil, fmt.Errorf("image download destination resolved to a non-public address")
		}
	}
	transport := base.Clone()
	transport.DisableKeepAlives = true
	if transport.Proxy != nil {
		proxyURL, err := transport.Proxy(req)
		if err != nil {
			return nil, err
		}
		if proxyURL != nil && req.URL.Scheme == "http" {
			return nil, fmt.Errorf("proxied image downloads require HTTPS")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.ServerName = host
	clone := req.Clone(req.Context())
	if clone.Host == "" {
		clone.Host = req.URL.Host
	}
	port := req.URL.Port()
	if port == "" {
		port = "443"
		if req.URL.Scheme == "http" {
			port = "80"
		}
	}
	clone.URL.Host = net.JoinHostPort(ips[0].IP.String(), port)
	// URL credentials must never become download authorization.
	if clone.URL.User != nil || strings.Contains(host, "%") {
		return nil, fmt.Errorf("image download credentials or scoped address are not allowed")
	}
	return transport.RoundTrip(clone)
}
