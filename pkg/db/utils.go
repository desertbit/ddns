package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

func Key(domain string, rtype uint16) (k string, err error) {
	if _, ok := dns.IsDomainName(domain); !ok {
		err = fmt.Errorf("invalid domain: %s", domain)
		return
	}

	// Ensure it is lower case only.
	domain = strings.ToLower(domain)

	// Reverse domain, starting from top-level domain
	// eg. "com.desertbit.test"
	labels := dns.SplitDomainName(domain)
	last := len(labels) - 1
	for i := 0; i < len(labels)/2; i++ {
		labels[i], labels[last-i] = labels[last-i], labels[i]
	}

	k = strings.Join(labels, ".") + "_" + strconv.Itoa(int(rtype))
	return
}
