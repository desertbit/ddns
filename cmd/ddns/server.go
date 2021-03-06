package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/desertbit/closer/v3"
	"github.com/desertbit/ddns/pkg/api"
	"github.com/desertbit/ddns/pkg/db"
	"github.com/desertbit/ddns/pkg/dns"
	"github.com/desertbit/grumble"
	"github.com/rs/zerolog/log"
)

func init() {
	App.AddCommand(&grumble.Command{
		Name: "server",
		Help: "start the server",
		Flags: func(f *grumble.Flags) {
			f.String("d", "db", "./ddns.db", "path to record database")
			f.String("c", "conf", "./ddns.yaml", "path to config file")
			f.StringL("dnsListenAddr", ":53", "dns udp listen address")
			f.StringL("apiListenAddr", ":80", "http api listen address")
		},
		Run: runServer,
	})
}

func runServer(ctx *grumble.Context) (err error) {
	cl := closer.New()
	defer cl.Close_()

	d, err := db.Open(cl, ctx.Flags.String("db"))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Start the api http server.
	as, err := api.NewServer(cl.CloserTwoWay(), ctx.Flags.String("apiListenAddr"), ctx.Flags.String("conf"), d)
	if err != nil {
		return fmt.Errorf("failed to create api server: %w", err)
	}
	as.Run()

	// Start the dns server.
	ds := dns.NewServer(cl.CloserTwoWay(), ctx.Flags.String("dnsListenAddr"), d)
	ds.Run()

	// Wait for closed or signal.
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-cl.ClosedChan():
	case <-sig:
	}
	log.Info().Msg("Closing")

	return
}
