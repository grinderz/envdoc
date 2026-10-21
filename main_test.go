package main

import (
	"flag"
	"os"
	"testing"

	"github.com/grinderz/envdoc/debug"
)

type testConfig struct {
	Debug bool
}

func TestMain(m *testing.M) {
	var cfg testConfig
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode")
	flag.Parse()

	debug.Config.Enabled = cfg.Debug

	os.Exit(m.Run())
}
