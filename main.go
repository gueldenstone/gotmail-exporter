package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/log/level"
	emailProber "github.com/gueldenstone/gotmail_exporter/internal/emailprober"
	"github.com/pborman/getopt/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
)

func init() {
	getopt.Parse()
	prometheus.MustRegister(version.NewCollector("gotmail_exporter"))
}

var (
	logLevel  = getopt.StringLong("log", 'l', "info", "logLevel")
	logLevels = map[string]level.Option{
		"debug": level.AllowDebug(),
		"info":  level.AllowInfo(),
		"warn":  level.AllowWarn(),
		"error": level.AllowWarn(),
	}
)

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	logger = level.NewFilter(logger, logLevels[*logLevel])
	prober := emailProber.NewEmailProber()
	level.Info(logger).Log("msg", "Starting gotmail_exportert", "version", version.Info(), "logLevel", *logLevel)
	setLogLevel(*logLevel)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		prober.Handler(w, r, logger)
	})
	http.ListenAndServe(":2112", nil)
}

func setLogLevel(lvl string) error {
	switch strings.ToLower(lvl) {
	case "":
		level.AllowInfo()
	case "debug":
		level.AllowDebug()
	case "info":
		level.AllowInfo()
	case "warn":
		level.AllowWarn()
	case "error":
		level.AllowError()
	default:
		return fmt.Errorf("unkown log level: %s", lvl)
	}
	return nil
}
