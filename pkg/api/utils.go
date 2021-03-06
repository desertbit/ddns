package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/miekg/dns"
)

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return ip
}

func toValidDomain(domain string) (string, error) {
	// Validate and prepare domain.
	if _, ok := dns.IsDomainName(domain); !ok {
		return "", fmt.Errorf("invalid domain")
	}
	labels := dns.SplitDomainName(domain)
	if len(labels) == 0 {
		return "", fmt.Errorf("invalid domain")
	}
	return strings.Join(labels, "."), nil
}
