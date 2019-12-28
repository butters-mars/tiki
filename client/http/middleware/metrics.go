package middleware

import (
	"context"
	"fmt"
	"time"

	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
)

const (
	labelSrc    = "src"
	labelTgt    = "tgt"
	labelURI    = "uri"
	labelMethod = "method"
	labelCode   = "status"
	labelSrcIP  = "src_ip"
	labelTgtIP  = "tgt_ip"
)

var allLabels = []string{labelSrc, labelTgt, labelURI, labelMethod}
var allLabelsWithCode = []string{labelSrc, labelTgt, labelURI, labelMethod, labelCode}

var qps metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
	Namespace: "service",
	Subsystem: "api",
	Name:      "call",
	Help:      "Api call.",
}, allLabels)

var errCode metrics.Counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
	Namespace: "service",
	Subsystem: "api",
	Name:      "errcode",
	Help:      "Error Code of api call.",
}, allLabelsWithCode)

var lantency metrics.Histogram = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
	Namespace: "service",
	Subsystem: "api",
	Name:      "latency",
	Help:      "Lantency of api call.",
}, allLabels)

// InitMetrics inits prometheus setting and starts server on given port
func InitMetrics() {
	initHystrixMetrics()
}

// Metrics returns a middleware to export metrics to prometheus
func Metrics(src, tgt, uri, method string) endpoint.Middleware {
	labels := []string{labelSrc, src, labelTgt, tgt, labelURI, uri, labelMethod, method}
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {

			var resp interface{}
			defer func(t time.Time) {
				lantency.With(labels...).Observe(time.Since(t).Seconds() * 1000) // millisecond
				qps.With(labels...).Add(1)
			}(time.Now())

			resp, err := next(ctx, request)

			if arr, ok := resp.([]interface{}); ok {
				if err != nil {
					logger.Infof("[Metrics] resp: %v, err: %v", arr[1], err)
				}
				if code, ok := arr[1].(int); ok && code >= 400 {
					errLabels := []string{}
					errLabels = append(errLabels, labels...)
					errLabels = append(errLabels, labelCode, fmt.Sprintf("%d", code))
					errCode.With(errLabels...).Add(1)
				}
			}

			return resp, err
		}
	}

}
