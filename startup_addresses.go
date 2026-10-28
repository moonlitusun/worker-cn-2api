package main

import (
	"bytes"
	"net"
	"sort"
	"strconv"
	"strings"
)

// startupAddresses resolves wildcard listeners to usable IPv4 URLs. Enumerating
// active interfaces avoids an external network probe and works offline on Windows.
func startupAddresses(host string, port int) []string {
	ips := []net.IP{}
	if interfaces, err := net.Interfaces(); err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addresses, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, address := range addresses {
				switch a := address.(type) {
				case *net.IPNet:
					ips = append(ips, a.IP)
				case *net.IPAddr:
					ips = append(ips, a.IP)
				}
			}
		}
	}
	return formatStartupAddresses(host, port, ips)
}

func formatStartupAddresses(host string, port int, ips []net.IP) []string {
	host = strings.Trim(host, "[]")
	if host != "" && host != "0.0.0.0" && host != "::" {
		return []string{net.JoinHostPort(host, strconv.Itoa(port))}
	}
	valid := []net.IP{}
	seen := map[string]bool{}
	for _, ip := range ips {
		ipv4 := ip.To4()
		if ipv4 == nil || !ipv4.IsGlobalUnicast() || ipv4.IsLinkLocalUnicast() || seen[ipv4.String()] {
			continue
		}
		seen[ipv4.String()] = true
		valid = append(valid, ipv4)
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].IsPrivate() != valid[j].IsPrivate() {
			return valid[i].IsPrivate()
		}
		return bytes.Compare(valid[i], valid[j]) < 0
	})
	if len(valid) == 0 {
		valid = append(valid, net.IPv4(127, 0, 0, 1))
	}
	addresses := make([]string, 0, len(valid))
	for _, ip := range valid {
		addresses = append(addresses, net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	}
	return addresses
}
