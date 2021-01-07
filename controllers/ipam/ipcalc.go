package ipam

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/mikioh/ipaddr"
)

func ParseIPRange(allocation string) ([]*net.IPNet, error) {
	if !strings.Contains(allocation, "-") {
		_, n, err := net.ParseCIDR(allocation)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q", allocation)
		}
		return []*net.IPNet{n}, nil
	}

	fs := strings.SplitN(allocation, "-", 2)
	if len(fs) != 2 {
		return nil, fmt.Errorf("invalid IP range %q", allocation)
	}
	start := net.ParseIP(strings.TrimSpace(fs[0]))
	if start == nil {
		return nil, fmt.Errorf("invalid IP range %q: invalid start IP %q", allocation, fs[0])
	}
	end := net.ParseIP(strings.TrimSpace(fs[1]))
	if end == nil {
		return nil, fmt.Errorf("invalid IP range %q: invalid end IP %q", allocation, fs[1])
	}

	var ret []*net.IPNet
	for _, pfx := range ipaddr.Summarize(start, end) {
		n := &net.IPNet{
			IP:   pfx.IP,
			Mask: pfx.Mask,
		}
		ret = append(ret, n)
	}
	return ret, nil
}

func GetIPs(cidr string) []string {

	var ips []string

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatal(err)
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	return ips
}

/*
func main() {
	cidr, _ := ParseIPRange("192.168.122.100/30")
	for _, item := range cidr {
		for _, ip := range GetIPs(item.String()) {
			fmt.Println(ip)
		}
	}
}
*/
