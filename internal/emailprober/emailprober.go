package emailProber

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type EmailProber struct {
	verifier *emailverifier.Verifier
}

func NewEmailProber() *EmailProber {
	v := emailverifier.NewVerifier()
	v.EnableSMTPCheck()
	return &EmailProber{
		verifier: v,
	}
}

func (p *EmailProber) Handler(w http.ResponseWriter, r *http.Request, logger log.Logger) {
	params := r.URL.Query()

	timeoutSeconds, err := getTimeout(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse timeout from Prometheus header: %s", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
	defer cancel()
	r = r.WithContext(ctx)

	probeSuccessGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_success",
		Help: "Displays whether or not the probe was a success",
	})
	probeDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})

	target := params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", http.StatusBadRequest)
		return
	}

	sl := newScrapeLogger(logger, target)
	level.Info(sl).Log("msg", "Beginning probe", "target", target, "timeout_seconds", timeoutSeconds)

	start := time.Now()
	registry := prometheus.NewRegistry()
	registry.MustRegister(probeSuccessGauge)
	registry.MustRegister(probeDurationGauge)
	err = p.probeEmail(ctx, target, registry, sl)
	duration := time.Since(start).Seconds()
	probeDurationGauge.Set(duration)
	if err == nil {
		probeSuccessGauge.Set(1)
		level.Info(sl).Log("msg", "Probe succeeded", "duration_seconds", duration)
	} else {
		level.Error(sl).Log("msg", "Probe failed", "error", err, "duration_seconds", duration)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func getTimeout(r *http.Request) (timeoutSeconds float64, err error) {
	// If a timeout is configured via the Prometheus header, add it to the request.
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		var err error
		timeoutSeconds, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 120
	}
	return timeoutSeconds, nil
}

type scrapeLogger struct {
	next         log.Logger
	buffer       bytes.Buffer
	bufferLogger log.Logger
}

func newScrapeLogger(logger log.Logger, target string) *scrapeLogger {
	logger = log.With(logger, "target", target)
	sl := &scrapeLogger{
		next:   logger,
		buffer: bytes.Buffer{},
	}
	bl := log.NewLogfmtLogger(&sl.buffer)
	sl.bufferLogger = log.With(bl, "ts", log.DefaultTimestampUTC, "caller", log.Caller(6), "target", target)
	return sl
}

func (sl scrapeLogger) Log(keyvals ...interface{}) error {
	sl.bufferLogger.Log(keyvals...)
	kvs := make([]interface{}, len(keyvals))
	copy(kvs, keyvals)
	// Switch level to debug for application output.
	for i := 0; i < len(kvs); i += 2 {
		if kvs[i] == level.Key() {
			kvs[i+1] = level.DebugValue()
		}
	}
	return sl.next.Log(kvs...)
}

func (p *EmailProber) probeEmail(ctx context.Context, target string, registry *prometheus.Registry, _ log.Logger) error {
	var (
		probeEmailReachable = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_email_reachable",
			Help: "Indicates if email is reachable",
		})
	)
	registry.MustRegister(probeEmailReachable)
	res, err := p.verifier.Verify(target)
	if err != nil {
		return fmt.Errorf("could not verify email %s: %w", target, err)
	}

	probeEmailReachable.Set(func() float64 {
		if res.Reachable == "yes" {
			return 1
		} else {
			return 0
		}
	}())
	return nil
}
