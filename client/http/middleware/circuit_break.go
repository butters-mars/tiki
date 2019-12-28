package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/afex/hystrix-go/hystrix/metric_collector"
	"github.com/go-kit/kit/endpoint"
)

// DefaultCBConfig provides default circuitbreak setting
var DefaultCBConfig = hystrix.CommandConfig{
	Timeout:                5000, // 5s
	MaxConcurrentRequests:  500,
	ErrorPercentThreshold:  50,
	SleepWindow:            5000, // 5s
	RequestVolumeThreshold: 20,
}

type hystrixLogger struct {
}

func (l hystrixLogger) Printf(format string, items ...interface{}) {
	logger.Infof(format, items...)
}

func initHystrixMetrics() {
	hystrix.SetLogger(hystrixLogger{})

	pc := NewPrometheusCollector(nil, nil)
	metricCollector.Registry.Register(pc.Collector)
}

var (
	configed = make(map[string]string)
	cmdMap   = make(map[string]string)
	mutex    = sync.RWMutex{}
)

// CircuitBreaker provides hystrix circuitbreaker for HTTP calls.
func CircuitBreaker(commandName string, commandCfg hystrix.CommandConfig) endpoint.Middleware {
	//hystrix.ConfigureCommand(commandName, commandCfg)

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// config endpoint level command
			addr := "0.0.0.0"
			if req, ok := request.(*http.Request); ok {
				addr = req.Host
			}
			cmd := fmt.Sprintf("%s-%s", commandName, addr)
			segs := strings.Split(commandName, "-")
			uri := segs[2]
			method := segs[3]
			key := fmt.Sprintf("%s-%s-%s", addr, uri, method)
			configCmd(cmd, key, commandCfg)

			var resp interface{}
			if err := hystrix.Do(cmd, func() (err error) {
				resp, err = next(ctx, request)
				return err
			}, nil); err != nil {
				return nil, err
			}
			return resp, nil
		}
	}
}

func configCmd(cmd string, key string, cfg hystrix.CommandConfig) {
	mutex.RLock()
	if _, ok := configed[cmd]; ok {
		mutex.RUnlock()
		return
	}
	mutex.RUnlock()

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := configed[cmd]; ok {
		return
	}
	hystrix.ConfigureCommand(cmd, cfg)
	configed[cmd] = key
	cmdMap[key] = cmd
	logger.Infof("[CB] endpoint %s -> %s configured", key, cmd)
}

// CleanupEndpoint cleans up endpoint info by given key(format: <addr>-<uri>-<method>)
func CleanupEndpoint(key string) {
	mutex.Lock()
	defer mutex.Unlock()

	cmd, ok := cmdMap[key]
	logger.Infof("[CB] endpoint %s -> %s(%v) cleaned up", key, cmd, ok)

	if ok {
		delete(configed, cmd)
	}
	delete(cmdMap, key)
}

// IsCircuitOpen return whether circuit of given key(format: <addr>-<uri>-<method>) is open
func IsCircuitOpen(key string) (open, ok bool) {
	mutex.RLock()
	cmd, _ok := cmdMap[key]
	mutex.RUnlock()

	if _ok {
		c, _, err := hystrix.GetCircuit(cmd)
		if err != nil {
			logger.Warnf("Fail to GetCircuit(%s): %v", cmd, err)
			return false, false
		}

		open = c.IsOpen()
		ok = true
	}

	return
}
