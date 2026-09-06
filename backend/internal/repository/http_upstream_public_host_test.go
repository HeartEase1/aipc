package repository

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicHostTransportPinsDNSAndPreservesHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != "images.example" {
			t.Errorf("Host = %q", req.Host)
		}
		_, _ = w.Write([]byte("image"))
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)
	dials := 0
	base := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		dials++
		if address != "8.8.8.8:80" {
			t.Errorf("destination was resolved again: %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
	}}
	transport := &publicHostTransport{base: base, lookup: func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}}
	req, err := http.NewRequest(http.MethodGet, "http://images.example/image.png", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, 1, dials)
	require.Equal(t, "images.example", req.URL.Host, "the caller's request must not be mutated")
	transport.lookup = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}}, nil
	}
	_, err = transport.RoundTrip(req)
	require.ErrorContains(t, err, "non-public")
	require.Equal(t, 1, dials, "a mixed public/private DNS answer must never reach the dialer")
}

func TestPublicHostTransportRejectsPrivateRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "http://127.0.0.1/private", http.StatusFound)
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)
	dials := 0
	base := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		dials++
		return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
	}}
	client := &http.Client{Transport: &publicHostTransport{base: base, lookup: func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}}}
	_, err := client.Get("http://images.example/start")
	require.ErrorContains(t, err, "not public")
	require.Equal(t, 1, dials)
}
