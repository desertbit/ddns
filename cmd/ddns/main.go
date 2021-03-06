package main

import (
	"os"

	"github.com/desertbit/grumble"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	flagVerbose = "verbose"
	flagJSON    = "json"
)

// Create the grumble app.
var App = grumble.New(&grumble.Config{
	Name:        "ddns",
	Description: "Dynamic DNS Service",

	Flags: func(f *grumble.Flags) {
		f.Bool("v", flagVerbose, false, "verbose mode")
		f.Bool("j", flagJSON, false, "JSON log mode")
	},
})

func init() {
	App.OnInit(func(a *grumble.App, f grumble.FlagMap) error {
		if !f.Bool(flagJSON) {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}

		if f.Bool(flagVerbose) {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		return nil
	})
}

func main() {
	grumble.Main(App)
}
