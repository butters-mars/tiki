package grpc

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"

	"github.com/tiki/config"
	"github.com/tiki/logging"
)

var logger = logging.Logger

type grpcClient struct {
}

func NewClientConn(address string, cfg config.ServiceDiscoveryCfg) (*grpc.ClientConn, error) {
	options := DialOptions(address, cfg)
	return grpc.Dial(address, options...)
}

func DialOptions(address string, cfg config.ServiceDiscoveryCfg) []grpc.DialOption {
	logEntry := logrus.NewEntry(logger)

	var r naming.Resolver
	if cfg.Type == "consul" {
		r = newConsulResolver(cfg.Consul)
	} else {
		r = newDirectResolver(address)
	}
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBalancer(grpc.RoundRobin(r)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_logrus.StreamClientInterceptor(logEntry),
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_logrus.UnaryClientInterceptor(logEntry),
		)),
	}
}
