package main

import (
	"flag"
	"net/http"

	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"

	consulapi "github.com/hashicorp/consul/api"

	fmgrpc "github.com/butters-mars/tiki/client/grpc"
	gw "github.com/butters-mars/tiki/example/svcdef"
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	consulCfg := &consulapi.Config{
		Address:    "localhost:8500",
		Datacenter: "dc1",
	}
	opts := fmgrpc.DialOptions(consulCfg)

	var err error
	err = gw.RegisterMathHandlerFromEndpoint(ctx, mux, "math.svc", opts)
	if err != nil {
		return err
	}

	opts = fmgrpc.DialOptions(consulCfg)
	err = gw.RegisterUserHandlerFromEndpoint(ctx, mux, "user.svc", opts)
	if err != nil {
		return err
	}

	opts = fmgrpc.DialOptions(consulCfg)
	err = gw.RegisterStringHandlerFromEndpoint(ctx, mux, "str.svc", opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8080", mux)
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := run(); err != nil {
		glog.Fatal(err)
	}
}
