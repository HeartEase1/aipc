package web

import "testing"

func TestIsMainlandChinaIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "China IPv4", ip: "114.114.114.114", want: true},
		{name: "China IPv4 second provider", ip: "223.5.5.5", want: true},
		{name: "China IPv6", ip: "2400:3200::1", want: true},
		{name: "foreign IPv4", ip: "8.8.8.8", want: false},
		{name: "foreign IPv6", ip: "2001:4860:4860::8888", want: false},
		{name: "private IPv4", ip: "192.168.1.10", want: false},
		{name: "loopback", ip: "::1", want: false},
		{name: "invalid", ip: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMainlandChinaIP(tt.ip); got != tt.want {
				t.Fatalf("isMainlandChinaIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
