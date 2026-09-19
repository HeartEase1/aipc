package web

import (
	_ "embed"
	"fmt"
	"net/netip"
	"strings"
)

//go:generate go run ../../cmd/update-china-ip-ranges -out china_ip_ranges.txt

//go:embed china_ip_ranges.txt
var mainlandChinaIPRanges string

var mainlandChinaIPs = mustParseIPPrefixSet(mainlandChinaIPRanges)

type ipPrefixNode struct {
	children [2]*ipPrefixNode
	terminal bool
}

type ipPrefixSet struct {
	v4 *ipPrefixNode
	v6 *ipPrefixNode
}

func mustParseIPPrefixSet(raw string) *ipPrefixSet {
	set := &ipPrefixSet{v4: &ipPrefixNode{}, v6: &ipPrefixNode{}}
	for lineNumber, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			panic(fmt.Sprintf("invalid embedded mainland China IP prefix on line %d: %v", lineNumber+1, err))
		}
		set.add(prefix.Masked())
	}
	return set
}

func (s *ipPrefixSet) add(prefix netip.Prefix) {
	addr := prefix.Addr().Unmap()
	node := s.v6
	if addr.Is4() {
		node = s.v4
	}
	bytes := addr.AsSlice()
	for bitIndex := 0; bitIndex < prefix.Bits(); bitIndex++ {
		bit := (bytes[bitIndex/8] >> (7 - bitIndex%8)) & 1
		if node.children[bit] == nil {
			node.children[bit] = &ipPrefixNode{}
		}
		node = node.children[bit]
	}
	node.terminal = true
	// More-specific entries below a terminal prefix cannot change membership.
	node.children = [2]*ipPrefixNode{}
}

func (s *ipPrefixSet) contains(addr netip.Addr) bool {
	if s == nil || !addr.IsValid() {
		return false
	}
	addr = addr.Unmap()
	node := s.v6
	if addr.Is4() {
		node = s.v4
	}
	bytes := addr.AsSlice()
	for bitIndex := 0; bitIndex < addr.BitLen(); bitIndex++ {
		if node == nil {
			return false
		}
		if node.terminal {
			return true
		}
		bit := (bytes[bitIndex/8] >> (7 - bitIndex%8)) & 1
		node = node.children[bit]
	}
	return node != nil && node.terminal
}

func isMainlandChinaIP(rawIP string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(rawIP))
	return err == nil && mainlandChinaIPs.contains(addr)
}
