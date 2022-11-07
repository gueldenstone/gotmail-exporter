package main

import (
	"net/http"

	"github.com/go-kit/log/level"
	emailProber "github.com/gueldenstone/gotmail_exporter/internal/emailprober"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
)

func init() {
	prometheus.MustRegister(version.NewCollector("gotmail_exporter"))
}

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	prober := emailProber.NewEmailProber()
	level.Info(logger).Log("msg", "Starting gotmail_exportert", "version", version.Info())
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		prober.Handler(w, r, logger)
	})
	http.ListenAndServe(":2112", nil)
}
