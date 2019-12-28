package tracing

import (
	"io"

	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
	"github.com/tiki/logging"
)

var logger = logging.Logger

// Init inits jaeger tracing with given appname and configuration
func Init(appname string, cfg *jaegercfg.Configuration) (closer io.Closer, err error) {
	if cfg == nil {
		cfg = &jaegercfg.Configuration{}
	}
	logger.Infof("Init tracing: %v", cfg.Sampler)

	//jLogger := jaegerlog.NullLogger
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err = cfg.InitGlobalTracer(
		appname,
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		logger.Errorf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}

	return
}
