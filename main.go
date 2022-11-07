package main

import (
	"net/http"

	emailProber "github.com/gueldenstone/gotmail_exporter/internal/emailprober"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
)

func main() {
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)
	prober := emailProber.NewEmailProber()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		prober.Handler(w, r, logger)
	})
	http.ListenAndServe(":2112", nil)
}
