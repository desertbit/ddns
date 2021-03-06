package dns

import (
	"github.com/desertbit/closer/v3"
	"github.com/desertbit/ddns/pkg/db"
	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
)

type Server struct {
	closer.Closer

	d *db.DB
	s *dns.Server
}

func NewServer(cl closer.Closer, listenAddr string, d *db.DB) (s *Server) {
	s = &Server{
		Closer: cl,
		d:      d,
		s: &dns.Server{
			Net:  "udp",
			Addr: listenAddr,
		},
	}
	s.s.Handler = s
	s.OnClosing(func() error {
		return s.s.Shutdown()
	})
	return
}

func (s *Server) Run() {
	log.Info().Str("listenAddr", s.s.Addr).Msg("dns server running")
	go s.serve()
}

func (s *Server) serve() {
	defer s.Close_()

	err := s.s.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("dns ListenAndServe failed")
		return
	}
}

// ServeDNS implements the dns server handler.
func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		s.parseQuery(m)
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Warn().Err(err).Msg("failed to write response message")
	}
}

func (s *Server) parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
			log.Warn().Str("name", q.Name).Uint16("type", q.Qtype).Msg("invalid request")
			continue
		}

		log.Debug().Str("name", q.Name).Uint16("type", q.Qtype).Msg("dns query")

		rr, err := s.getRecord(q.Name, q.Qtype)
		if err != nil {
			if err == db.ErrNotFound {
				log.Debug().Str("name", q.Name).Uint16("type", q.Qtype).Msg("no record found")
			} else {
				log.Error().Err(err).Str("name", q.Name).Uint16("type", q.Qtype).Msg("failed to get record")
			}
			continue
		}

		m.Answer = append(m.Answer, rr)
	}
}

func (s *Server) getRecord(domain string, rtype uint16) (rr dns.RR, err error) {
	k, err := db.Key(domain, rtype)
	if err != nil {
		return
	}

	r, err := s.d.GetRecord(k)
	if err != nil {
		return
	}

	return dns.NewRR(r.RR)
}
