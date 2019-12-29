package grpc

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"github.com/butters-mars/tiki/config"
	"github.com/butters-mars/tiki/logging"
)

var (
	logger = logging.L
)

// NewServer creates a grpc server with middlewares setup
func NewServer(logEntry *logrus.Entry, auth grpc_auth.AuthFunc, authCfg *config.AuthConfig) *grpc.Server {
	if logEntry == nil {
		logEntry = logrus.NewEntry(logger)
	}

	if auth == nil {
		auth = func(ctx context.Context) (context.Context, error) {
			return ctx, nil
		}
	}

	srvOpts := make([]grpc.ServerOption, 0)
	if authCfg != nil && authCfg.TLS {
		logger.Infof("[grpc] using TLS for server")
		creds, _ := credentials.NewServerTLSFromFile(authCfg.CertFile, authCfg.KeyFile)
		srvOpts = append(srvOpts, grpc.Creds(creds))
	}

	srvOpts = append(srvOpts,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_opentracing.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_logrus.StreamServerInterceptor(logEntry),
			grpc_auth.StreamServerInterceptor(auth),
			grpc_recovery.StreamServerInterceptor(),
			grpc_validator.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(logEntry),
			grpc_auth.UnaryServerInterceptor(auth),
			grpc_recovery.UnaryServerInterceptor(),
			grpc_validator.UnaryServerInterceptor(),
		)),
	)
	return grpc.NewServer(srvOpts...)
}

// EnableHandlingTiming enables client/server handling timing with prometheus
func EnableHandlingTiming() {
	grpc_prometheus.EnableClientHandlingTimeHistogram()
	grpc_prometheus.EnableHandlingTimeHistogram()
}
