// Command update-china-ip-ranges refreshes the embedded mainland China IP
// allocation list from APNIC's delegated statistics.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/bits"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const sourceURL = "https://ftp.apnic.net/stats/apnic/delegated-apnic-latest"

func main() {
	outPath := flag.String("out", "china_ip_ranges.txt", "output file")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sourceURL)
	if err != nil {
		fatalf("download APNIC delegated data: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fatalf("download APNIC delegated data: HTTP %d", resp.StatusCode)
	}

	prefixes, serial, err := parseDelegated(resp.Body)
	if err != nil {
		fatalf("parse APNIC delegated data: %v", err)
	}
	if len(prefixes) == 0 {
		fatalf("parse APNIC delegated data: no CN prefixes found")
	}

	file, err := os.Create(*outPath)
	if err != nil {
		fatalf("create output: %v", err)
	}
	w := bufio.NewWriter(file)
	_, _ = fmt.Fprintf(w, "# Generated from %s\n# APNIC serial: %s\n", sourceURL, serial)
	for _, prefix := range prefixes {
		_, _ = fmt.Fprintln(w, prefix.String())
	}
	if err := w.Flush(); err != nil {
		_ = file.Close()
		fatalf("write output: %v", err)
	}
	if err := file.Close(); err != nil {
		fatalf("close output: %v", err)
	}
	fmt.Printf("wrote %d mainland China prefixes to %s (APNIC serial %s)\n", len(prefixes), *outPath, serial)
}

func parseDelegated(r io.Reader) ([]netip.Prefix, string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	serial := "unknown"
	prefixes := make([]netip.Prefix, 0, 10000)
	seen := make(map[netip.Prefix]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 6 && parts[0] == "2" && parts[1] == "apnic" {
			serial = parts[5]
			continue
		}
		if len(parts) < 7 || parts[0] != "apnic" || parts[1] != "CN" {
			continue
		}
		status := parts[6]
		if status != "allocated" && status != "assigned" {
			continue
		}

		var rows []netip.Prefix
		switch parts[2] {
		case "ipv4":
			count, err := strconv.ParseUint(parts[4], 10, 64)
			if err != nil {
				return nil, serial, fmt.Errorf("invalid IPv4 count in %q: %w", line, err)
			}
			rows, err = ipv4RangeToPrefixes(parts[3], count)
			if err != nil {
				return nil, serial, fmt.Errorf("invalid IPv4 allocation %q: %w", line, err)
			}
		case "ipv6":
			addr, err := netip.ParseAddr(parts[3])
			if err != nil {
				return nil, serial, fmt.Errorf("invalid IPv6 address in %q: %w", line, err)
			}
			prefixBits, err := strconv.Atoi(parts[4])
			if err != nil || prefixBits < 0 || prefixBits > 128 {
				return nil, serial, fmt.Errorf("invalid IPv6 prefix length in %q", line)
			}
			rows = []netip.Prefix{netip.PrefixFrom(addr, prefixBits).Masked()}
		default:
			continue
		}

		for _, prefix := range rows {
			if _, ok := seen[prefix]; ok {
				continue
			}
			seen[prefix] = struct{}{}
			prefixes = append(prefixes, prefix)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, serial, err
	}

	sort.Slice(prefixes, func(i, j int) bool {
		left, right := prefixes[i], prefixes[j]
		if left.Addr().BitLen() != right.Addr().BitLen() {
			return left.Addr().BitLen() < right.Addr().BitLen()
		}
		if left.Addr() != right.Addr() {
			return left.Addr().Less(right.Addr())
		}
		return left.Bits() < right.Bits()
	})
	return prefixes, serial, nil
}

func ipv4RangeToPrefixes(rawAddress string, count uint64) ([]netip.Prefix, error) {
	parsed := net.ParseIP(rawAddress).To4()
	if parsed == nil || count == 0 || count > 1<<32 {
		return nil, fmt.Errorf("invalid address or count")
	}
	start := uint64(parsed[0])<<24 | uint64(parsed[1])<<16 | uint64(parsed[2])<<8 | uint64(parsed[3])
	if start+count > 1<<32 {
		return nil, fmt.Errorf("range exceeds IPv4 address space")
	}

	prefixes := make([]netip.Prefix, 0, 4)
	for count > 0 {
		alignment := uint64(1) << 32
		if start != 0 {
			alignment = uint64(1) << min(bits.TrailingZeros64(start), 32)
		}
		block := uint64(1) << (bits.Len64(count) - 1)
		if alignment < block {
			block = alignment
		}
		prefixBits := 32 - (bits.Len64(block) - 1)
		addr := netip.AddrFrom4([4]byte{byte(start >> 24), byte(start >> 16), byte(start >> 8), byte(start)})
		prefixes = append(prefixes, netip.PrefixFrom(addr, prefixBits))
		start += block
		count -= block
	}
	return prefixes, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
