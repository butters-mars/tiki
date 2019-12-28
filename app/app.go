package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc/grpclog"

	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	fmgrpc "github.com/tiki/client/grpc"
	fmhttp "github.com/tiki/client/http"
	"github.com/tiki/config"
	fmsgrpc "github.com/tiki/grpc"
	"github.com/tiki/healthcheck"
	"github.com/tiki/logging"
	"github.com/tiki/sd"
	"github.com/tiki/tracing"
	"github.com/tiki/utils"
)

// App is the main entry of an application, which provides
// setup of the framework based on configuration
type App struct {
	cfgName    string
	cfg        *config.Config
	registrars []func(base *grpc.Server)
	// auth see: https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth
	authFunc grpc_auth.AuthFunc
	//AuthService grpc_auth.ServiceAuthFuncOverride

	LogEntry      *logrus.Entry
	tracingCloser io.Closer
}

// GRPCRegistrar provides a way to register grpc server to the base server
//type GRPCRegistrar func(base *grpc.Server)

const (
	//debugAddr  = "debugaddr"
	//httpAddr   = "httpaddr"
	appName    = "appname"
	port       = "port"
	cfgTracing = "tracing"
	cfgAuth    = "auth"
	cfgSD      = "service-discovery"

	configName        = "config"
	samplingServerURL = "http://127.0.0.1:5778/sampling"
	localAgent        = "127.0.0.1:6831"
)

var logger = logging.Logger

// New creates an application instance
func New(cfgName string) App {
	if cfgName == "" {
		cfgName = configName
	}
	app := App{
		cfgName:    cfgName,
		registrars: make([]func(base *grpc.Server), 0),
		cfg:        initConfig(cfgName),
	}

	initMetrics()
	initHealthcheck()
	grpclog.SetLogger(logging.Logger)
	logger.Infof("setup tracing")
	closer, err := tracing.Init(app.cfg.APPName, app.cfg.Tracing)
	if err != nil {
		logger.Errorf("Fail to init tracing: %v", err)
	} else {
		app.tracingCloser = closer
	}

	discInfo := ""
	consulCfg := app.cfg.ServiceDiscovery.Consul
	if consulCfg != nil {
		discInfo = fmt.Sprintf("consul::%s/%s", consulCfg.Address, consulCfg.Datacenter)
	}
	fmhttp.SetupClient(app.cfg.APPName, app.cfg.UpstreamSetting, discInfo)

	return app
}

// SetAuthFunc set the auth function
func (app App) SetAuthFunc(authFunc func(context.Context) (context.Context, error)) {
	app.authFunc = authFunc
}

//RegisterGRPCServer add an registrar, and will do registration when app starts
func (app App) RegisterGRPCServer(registrar func(base *grpc.Server)) {
	if registrar == nil {
		return
	}

	app.registrars = append(app.registrars, registrar)
}

// NewGRPCConn creates grpc client conn from given address
func (app App) NewGRPCConn(addr string) (*grpc.ClientConn, error) {
	return fmgrpc.NewClientConn(addr, app.cfg.ServiceDiscovery)
}

// GetConfigProps return properties defined in app config
func (app App) GetConfigProps() map[string]string {
	return app.cfg.Properties
}

// Start starts the application
func (app App) Start() {
	if app.tracingCloser != nil {
		defer app.tracingCloser.Close()
	}

	var g run.Group

	// init debug handler
	ip := utils.GetIP()
	port := app.cfg.Port
	debugPort := port - 2000
	debugAddr := fmt.Sprintf(":%d", debugPort)
	debugListener, err := net.Listen("tcp", debugAddr)
	if err != nil {
		logger.Info("transport", "debug/HTTP", "during", "Listen", "err", err)
		os.Exit(1)
	}
	g.Add(func() error {
		logger.Info("transport", "debug/HTTP", "addr", debugAddr)
		return http.Serve(debugListener, http.DefaultServeMux)
	}, func(error) {
		debugListener.Close()
	})

	// The HTTP listener mounts the Go kit HTTP handler we created.
	/*
		httpAddr := app.cfg.HTTPAddr
		httpListener, err := net.Listen("tcp", httpAddr)
		if err != nil {
			logger.Info("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Info("transport", "HTTP", "addr", httpAddr)
			return http.Serve(httpListener, nil)
		}, func(error) {
			httpListener.Close()
		})
	*/

	// The gRPC listener mounts the Go kit gRPC server we created.
	grpcAddr := fmt.Sprintf(":%d", port)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Info("transport", "gRPC", "during", "Listen", "err", err)
		os.Exit(1)
	}
	g.Add(func() error {
		logger.Info("transport", "gRPC", "addr", grpcAddr)
		baseServer := fmsgrpc.NewServer(app.LogEntry, app.authFunc, app.cfg.Auth)
		for _, reg := range app.registrars {
			reg(baseServer)
		}

		return baseServer.Serve(grpcListener)
	}, func(error) {
		grpcListener.Close()
	})

	// Register to consul
	reg, svc, err := sd.InitServiceDiscovery(&sd.ServiceDiscoverySt{
		Type:          "consul",
		SvcName:       app.cfg.APPName,
		CheckEndpoint: "/healthcheck",
		CheckAddr:     fmt.Sprintf("%s:%d", ip, port-2000),
	}, fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		logger.Errorf("Fail to reg to consul: %v", err)
	} else {
		defer reg.Unregister(svc)
	}

	// This function just sits and waits for ctrl-C.
	cancelInterrupt := make(chan struct{})
	g.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return fmt.Errorf("received signal %s", sig)
		case <-cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})

	logger.Info("exit", g.Run())

}

func initConfig(cfgName string) *config.Config {
	cfgViper := viper.New()

	// set default values
	cfgViper.SetDefault(appName, "")
	cfgViper.SetDefault(port, 8080)
	cfgViper.SetDefault(cfgTracing,
		&jaegercfg.Configuration{
			Sampler: &jaegercfg.SamplerConfig{
				Type:  jaeger.SamplerTypeProbabilistic,
				Param: 1.0,
			},
		},
	)
	cfgViper.SetDefault(cfgAuth, &config.AuthConfig{})
	cfgViper.SetDefault(cfgSD, config.ServiceDiscoveryCfg{
		Type: "direct",
	})

	cfgViper.SetConfigName(cfgName) // name of config file (without extension)
	cfgViper.AddConfigPath(".")     // optionally look for config in the working directory
	err := cfgViper.ReadInConfig()  // Find and read the config file
	if err != nil {
		logger.Warnf("Fail to find config %s, err: %v", configName, err)
	}

	// support env checking
	cfgViper.SetEnvPrefix("TIT")
	cfgViper.AutomaticEnv()

	cfg := &config.Config{}
	cfgViper.Unmarshal(cfg)

	// unmarshal not work, setup mannually
	props := cfgViper.GetStringMapString("props")
	cfg.Properties = props

	logger.WithField("cfg", cfg).Info("setup app config")
	return cfg
}

func initMetrics() {
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())
	fmsgrpc.EnableHandlingTiming()
	logger.Info("setup metrics handler /metrics")
}

func initHealthcheck() {
	http.DefaultServeMux.Handle("/healthcheck", healthcheck.Handler())
	logger.Info("setup healthcheck /healthcheck")
}
