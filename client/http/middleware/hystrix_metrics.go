package middleware

import (
	"fmt"
	"strings"
	"time"

	metricCollector "github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tiki/utils"
)

// PrometheusNamespace namespace for hystrix metrics
const PrometheusNamespace = "hystrix_go"

// PrometheusCollector struct contains the metrics for prometheus. The handling of the values is completely done by the prometheus client library.
// The function `Collector` can be registered to the metricsCollector.Registry.
// If one want to use a custom registry it can be given via the reg parameter. If reg is nil, the prometheus default
// registry is used.
// The RunDuration is observed via a prometheus histogram ( https://prometheus.io/docs/concepts/metric_types/#histogram ).
// If the duration_buckets slice is nil, the "github.com/prometheus/client_golang/prometheus".DefBuckets  are used. As stated by the prometheus documentation, one should
// tailor the buckets to the response times of your application.
//
//
// Example use
//  package main
//
//  import (
//  	"github.com/afex/hystrix-go/plugins"
//  	"github.com/afex/hystrix-go/hystrix/metric_collector"
//  )
//
//  func main() {
//  	pc := plugins.NewPrometheusCollector(nil, nil)
//  	metricCollector.Registry.Register(pc.Collector)
//  }
type PrometheusCollector struct {
	attempts                *prometheus.CounterVec
	errors                  *prometheus.CounterVec
	successes               *prometheus.CounterVec
	failures                *prometheus.CounterVec
	rejects                 *prometheus.CounterVec
	shortCircuits           *prometheus.CounterVec
	timeouts                *prometheus.CounterVec
	fallbackSuccesses       *prometheus.CounterVec
	fallbackFailures        *prometheus.CounterVec
	contextCanceled         *prometheus.CounterVec
	contextDeadlineExceeded *prometheus.CounterVec
	totalDuration           *prometheus.GaugeVec
	runDuration             *prometheus.HistogramVec
	concurrencyInUse        *prometheus.HistogramVec
}

var (
	labels = []string{"command", "method", "uri", "src", "tgt", "src_ip", "tgt_ip"}
	ip     = "0.0.0.0"
)

func init() {
	ip = utils.GetIP()
	logger.Infof("local ip is %s", ip)
}

// NewPrometheusCollector creates collector
func NewPrometheusCollector(reg prometheus.Registerer, durationBuckets []float64) PrometheusCollector {
	logger.Infof("creating prometheus collector for hystrix metrics ...")
	if durationBuckets == nil {
		durationBuckets = prometheus.DefBuckets
	}
	hm := PrometheusCollector{
		attempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "attempts",
			Help:      "The number of updates.",
		}, labels),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "errors",
			Help:      "The number of unsuccessful attempts. Attempts minus Errors will equal successes within a time range. Errors are any result from an attempt that is not a success.",
		}, labels),
		successes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "successes",
			Help:      "The number of requests that succeed.",
		}, labels),
		failures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "failures",
			Help:      "The number of requests that fail.",
		}, labels),
		rejects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "rejects",
			Help:      "The number of requests that are rejected.",
		}, labels),
		shortCircuits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "short_circuits",
			Help:      "The number of requests that short circuited due to the circuit being open.",
		}, labels),
		timeouts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "timeouts",
			Help:      "The number of requests that are timeouted in the circuit breaker.",
		}, labels),
		fallbackSuccesses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "fallback_successes",
			Help:      "The number of successes that occurred during the execution of the fallback function.",
		}, labels),
		fallbackFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "fallback_failures",
			Help:      "The number of failures that occurred during the execution of the fallback function.",
		}, labels),
		contextCanceled: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "context_canceled",
			Help:      "The number of contextCanceled that occurred during the execution of the fallback function.",
		}, labels),
		contextDeadlineExceeded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: PrometheusNamespace,
			Name:      "context_deadline_exceeded",
			Help:      "The number of contextDeadlineExceeded that occurred during the execution of the fallback function.",
		}, labels),
		totalDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: PrometheusNamespace,
			Name:      "total_duration_seconds",
			Help:      "The total runtime of this command in seconds.",
		}, labels),
		runDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: PrometheusNamespace,
			Name:      "run_duration_seconds",
			Help:      "Runtime of the Hystrix command.",
			Buckets:   durationBuckets,
		}, labels),
		concurrencyInUse: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: PrometheusNamespace,
			Name:      "concurrency_inuse",
			Help:      "Concurrency in use of the Hystrix command.",
			Buckets:   durationBuckets,
		}, labels),
	}
	if reg != nil {
		reg.MustRegister(
			hm.attempts,
			hm.errors,
			hm.failures,
			hm.rejects,
			hm.shortCircuits,
			hm.timeouts,
			hm.fallbackSuccesses,
			hm.fallbackFailures,
			hm.totalDuration,
			hm.runDuration,
			hm.concurrencyInUse,
			hm.contextCanceled,
			hm.contextDeadlineExceeded,
		)
	} else {
		prometheus.MustRegister(
			hm.attempts,
			hm.errors,
			hm.failures,
			hm.rejects,
			hm.shortCircuits,
			hm.timeouts,
			hm.fallbackSuccesses,
			hm.fallbackFailures,
			hm.totalDuration,
			hm.runDuration,
			hm.concurrencyInUse,
			hm.contextCanceled,
			hm.contextDeadlineExceeded,
		)
	}
	return hm
}

type cmdCollector struct {
	commandName string
	src         string
	tgt         string
	uri         string
	addr        string
	method      string
	metrics     *PrometheusCollector
}

//	{"command", "method", "uri", "src", "tgt", "src_ip", "tgt_ip"}
func (hc *cmdCollector) getLabelValues() []string {
	return []string{hc.commandName, hc.method, hc.uri, hc.src, hc.tgt, ip, hc.addr}
}

func (hc *cmdCollector) initCounters() {
	hc.metrics.attempts.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.errors.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.successes.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.failures.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.rejects.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.shortCircuits.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.timeouts.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.fallbackSuccesses.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.fallbackFailures.WithLabelValues(hc.getLabelValues()...).Add(0.0)
	hc.metrics.totalDuration.WithLabelValues(hc.getLabelValues()...).Set(0.0)
}

// Collector returns collector for a given command
func (hm *PrometheusCollector) Collector(name string) metricCollector.MetricCollector {
	strs := strings.Split(name, "-")
	src := strs[0]
	host := strs[1]
	uri := strs[2]
	method := strs[3]
	addr := strs[4]

	name = fmt.Sprintf("%s-%s", method, uri)

	hc := &cmdCollector{
		commandName: name,
		src:         src,
		tgt:         host,
		uri:         uri,
		method:      method,
		addr:        addr,
		metrics:     hm,
	}

	hc.initCounters()
	return hc
}

// Update updates hystrix metrics
func (hc *cmdCollector) Update(r metricCollector.MetricResult) {
	// if r.Successes > 0 {
	// 	g.setGauge(g.circuitOpenPrefix, 0)
	// } else if r.ShortCircuits > 0 {
	// 	g.setGauge(g.circuitOpenPrefix, 1)
	// }

	callNTimes(r.Attempts, hc.IncrementAttempts)
	callNTimes(r.Errors, hc.IncrementErrors)
	callNTimes(r.Successes, hc.IncrementSuccesses)
	callNTimes(r.Failures, hc.IncrementFailures)
	callNTimes(r.Rejects, hc.IncrementRejects)
	callNTimes(r.ShortCircuits, hc.IncrementShortCircuits)
	callNTimes(r.Timeouts, hc.IncrementTimeouts)
	callNTimes(r.FallbackSuccesses, hc.IncrementFallbackSuccesses)
	callNTimes(r.FallbackFailures, hc.IncrementFallbackFailures)
	callNTimes(r.ContextCanceled, hc.IncrementContextCanceled)
	callNTimes(r.ContextDeadlineExceeded, hc.IncrementContextDeadlineExceeded)

	hc.UpdateTotalDuration(r.TotalDuration)
	hc.UpdateRunDuration(r.RunDuration)
	hc.UpdateConcurrencyInUse(r.ConcurrencyInUse)
}

// IncrementAttempts increments the number of updates.
func (hc *cmdCollector) IncrementAttempts() {
	hc.metrics.attempts.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementErrors increments the number of unsuccessful attempts.
// Attempts minus Errors will equal successes within a time range.
// Errors are any result from an attempt that is not a success.
func (hc *cmdCollector) IncrementErrors() {
	hc.metrics.errors.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementSuccesses increments the number of requests that succeed.
func (hc *cmdCollector) IncrementSuccesses() {
	hc.metrics.successes.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementFailures increments the number of requests that fail.
func (hc *cmdCollector) IncrementFailures() {
	hc.metrics.failures.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementRejects increments the number of requests that are rejected.
func (hc *cmdCollector) IncrementRejects() {
	hc.metrics.rejects.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementShortCircuits increments the number of requests that short circuited due to the circuit being open.
func (hc *cmdCollector) IncrementShortCircuits() {
	hc.metrics.shortCircuits.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementTimeouts increments the number of timeouts that occurred in the circuit breaker.
func (hc *cmdCollector) IncrementTimeouts() {
	hc.metrics.timeouts.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementFallbackSuccesses increments the number of successes that occurred during the execution of the fallback function.
func (hc *cmdCollector) IncrementFallbackSuccesses() {
	hc.metrics.fallbackSuccesses.WithLabelValues(hc.getLabelValues()...).Inc()
}

// IncrementFallbackFailures increments the number of failures that occurred during the execution of the fallback function.
func (hc *cmdCollector) IncrementFallbackFailures() {
	hc.metrics.fallbackFailures.WithLabelValues(hc.getLabelValues()...).Inc()
}

func (hc *cmdCollector) IncrementContextCanceled() {
	hc.metrics.contextCanceled.WithLabelValues(hc.getLabelValues()...).Inc()
}

func (hc *cmdCollector) IncrementContextDeadlineExceeded() {
	hc.metrics.contextDeadlineExceeded.WithLabelValues(hc.getLabelValues()...).Inc()
}

// UpdateTotalDuration updates the internal counter of how long we've run for.
func (hc *cmdCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	hc.metrics.totalDuration.WithLabelValues(hc.getLabelValues()...).Set(timeSinceStart.Seconds())
}

// UpdateRunDuration updates the internal counter of how long the last run took.
func (hc *cmdCollector) UpdateRunDuration(runDuration time.Duration) {
	hc.metrics.runDuration.WithLabelValues(hc.getLabelValues()...).Observe(runDuration.Seconds())
}

// UpdateConcurrencyInUse updates concurrency num in use.
func (hc *cmdCollector) UpdateConcurrencyInUse(num float64) {
	hc.metrics.concurrencyInUse.WithLabelValues(hc.getLabelValues()...).Observe(num)
}

// Reset resets the internal counters and timers.
func (hc *cmdCollector) Reset() {
}

func callNTimes(n float64, f func()) {
	for i := 0; i < int(n); i++ {
		f()
	}
}
