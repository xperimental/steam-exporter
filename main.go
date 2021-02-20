package main

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/xperimental/steam-exporter/internal/collector"
	"github.com/xperimental/steam-exporter/internal/config"
	"net/http"
	"os"
)

var (
	Version   = ""
	GitCommit = ""

	log = logrus.New()
)

func main() {
	cfg, err := config.Get(os.Args[0], os.Args[1:])
	switch {
	case errors.Is(err, pflag.ErrHelp):
		return
	case err != nil:
		log.Fatalf("Error reading configuration: %s", err)
	}

	if cfg.Verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	c, err := collector.New(log, cfg.Servers, cfg.DataTimeout)
	if err != nil {
		log.Fatalf("Can not create collector: %s", err)
	}

	if err := prometheus.Register(c); err != nil {
		log.Fatalf("Can not register collector: %s", err)
	}

	http.Handle("/metrics", promhttp.Handler())

	log.Infof("Listening on %s ...", cfg.ListenAddress)
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		log.Fatalf("Error creating listener: %s", err)
	}
}
