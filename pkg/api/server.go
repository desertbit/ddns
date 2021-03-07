package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/desertbit/closer/v3"
	"github.com/desertbit/ddns/pkg/db"
	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
)

type Server struct {
	closer.Closer

	d *db.DB
	c *Config
	s *http.Server
}

func NewServer(cl closer.Closer, listenAddr, configPath string, d *db.DB) (s *Server, err error) {
	c, err := parseConfig(configPath)
	if err != nil {
		return
	}

	s = &Server{
		Closer: cl,
		d:      d,
		c:      c,
		s: &http.Server{
			Addr: listenAddr,
		},
	}
	s.s.Handler = s
	s.OnClosing(s.s.Close)
	return
}

func (s *Server) Run() {
	log.Info().Str("listenAddr", s.s.Addr).Msg("http server running")
	go s.s.ListenAndServe()
}

// ServeHTTP implements the http server handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Only allow post methods.
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON in the body.
	dec := json.NewDecoder(r.Body)
	var d Data
	err := dec.Decode(&d)
	if err != nil {
		log.Warn().Err(err).Msg("failed to decode json request")
		http.Error(w, "decode failed", http.StatusBadRequest)
		return
	}

	// Validate and prepare domain.
	domain, err := toValidDomain(d.Domain)
	if err != nil {
		log.Warn().Err(err).Msg("invalid domain")
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	// Check if access is granted for this domain.
	// The api service must run behind a secure HTTP proxy.
	if len(d.Key) < minKeyLen {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	dk, ok := s.c.DomainKeys[domain]
	if !ok {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	if subtle.ConstantTimeCompare([]byte(dk), []byte(d.Key)) != 1 {
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// Parse IP.
	cip := getClientIP(r)
	ip := net.ParseIP(cip)
	if ip == nil {
		log.Warn().Str("ip", cip).Msg("invalid ip")
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}

	// Check if this is a IPv4 or IPv6 request and prepare the record.
	var (
		rr    string
		rtype = dns.TypeA
		ip4   = ip.To4()
	)
	if ip4 != nil {
		rr = fmt.Sprintf("%s. %v IN A %s", domain, s.c.TTL, ip4.String())
	} else {
		ip6 := ip.To16()
		if ip6 == nil {
			http.Error(w, "invalid ip", http.StatusBadRequest)
			return
		}
		rtype = dns.TypeAAAA
		rr = fmt.Sprintf("%s. %v IN AAAA %s", domain, s.c.TTL, ip6.String())
	}

	k, err := db.Key(domain, rtype)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get db key")
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	err = s.d.SetRecord(k, db.Record{
		RR:      rr,
		Expires: time.Now().Add(s.c.TTL).Add(time.Minute).Unix(),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to save record")
		http.Error(w, "failed to save record", http.StatusInternalServerError)
		return
	}

	log.Info().Str("domain", domain).Msg("updated record")
}
