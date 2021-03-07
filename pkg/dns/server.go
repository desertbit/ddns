package dns

import (
	"github.com/desertbit/closer/v3"
	"github.com/desertbit/ddns/pkg/db"
	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
)

type Server struct {
	closer.Closer

	d  *db.DB
	us *dns.Server
	ts *dns.Server
}

func NewServer(cl closer.Closer, udpListenAddr, tcpListenAddr string, d *db.DB) (s *Server) {
	s = &Server{
		Closer: cl,
		d:      d,
		us: &dns.Server{
			Net:  "udp",
			Addr: udpListenAddr,
		},
		ts: &dns.Server{
			Net:  "tcp",
			Addr: tcpListenAddr,
		},
	}
	s.us.Handler = s
	s.ts.Handler = s
	s.OnClosing(s.us.Shutdown)
	s.OnClosing(s.ts.Shutdown)
	return
}

func (s *Server) Run() {
	log.Info().Str("listenAddr", s.us.Addr).Msg("udp dns server running")
	log.Info().Str("listenAddr", s.ts.Addr).Msg("tcp dns server running")
	go s.serveUDP()
	go s.serveTCP()
}

func (s *Server) serveUDP() {
	defer s.Close_()

	err := s.us.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("dns udp ListenAndServe failed")
		return
	}
}

func (s *Server) serveTCP() {
	defer s.Close_()

	err := s.ts.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("dns tcp ListenAndServe failed")
		return
	}
}

// ServeDNS implements the dns server handler.
func (s *Server) ServeDNS(w dns.ResponseWriter, request *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(request)
	m.Compress = false

	switch request.Opcode {
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
