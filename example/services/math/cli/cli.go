package main

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/grpc-ecosystem/go-grpc-prometheus"

	//consul "github.com/hashicorp/consul/api"

	"google.golang.org/grpc"

	"github.com/butters-mars/tiki/example/svcdef"
)

func main() {
	//cc := grpc.NewClient
	cc, err := grpc.Dial("localhost:3333",
		grpc.WithInsecure(),
		//grpc.WithBalancer(grpc.RoundRobin(r)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			//grpc_logrus.StreamClientInterceptor(logEntry),
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			//grpc_logrus.UnaryClientInterceptor(logEntry),
		)),
	)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	cli := svcdef.NewMathClient(cc)
	r, err := cli.Do(context.Background(), &svcdef.Args{A: 2, B: 3})
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	fmt.Printf("r: %v\n", r.V)
}
