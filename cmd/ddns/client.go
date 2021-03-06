package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/desertbit/closer/v3"
	"github.com/desertbit/ddns/pkg/api"
	"github.com/desertbit/grumble"
	"github.com/rs/zerolog/log"
)

func init() {
	App.AddCommand(&grumble.Command{
		Name: "client",
		Help: "start the client",
		Flags: func(f *grumble.Flags) {
			f.String("u", "url", "https://ns.sample.com", "url of destination dyndns server")
			f.String("d", "domain", "", "domain to update")
			f.String("k", "key", "", "domain access key")
			f.Duration("i", "interval", time.Minute, "update interval")
		},
		Run: runClient,
	})
}

func runClient(ctx *grumble.Context) (err error) {
	cl := closer.New()
	defer cl.Close_()

	if !strings.HasPrefix(ctx.Flags.String("url"), "https://") {
		return fmt.Errorf("url must start with https://")
	}

	go func() {
		var (
			closingChan = cl.ClosingChan()
			c           = api.NewClient(ctx.Flags.String("url"), ctx.Flags.String("domain"), ctx.Flags.String("key"))
		)

		t := time.NewTicker(ctx.Flags.Duration("interval"))
		defer t.Stop()

		for {
			gerr := c.Do()
			if gerr != nil {
				log.Error().Err(gerr).Msg("request failed")
			} else {
				log.Info().Msg("updated record")
			}

			select {
			case <-closingChan:
				return
			case <-t.C:
				continue
			}
		}
	}()

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
