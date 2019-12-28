package main

import (
	"context"
	"fmt"
	"os"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	//consul "github.com/hashicorp/consul/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/tiki/example/svcdef"
)

func main() {

	input := os.Args[1]

	//cc := grpc.NewClient
	cc, err := grpc.Dial("localhost:5444",
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

	md := metadata.Pairs("authorization", "Bearer XXXX")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	cli := svcdef.NewStringClient(cc)
	r, err := cli.Reverse(ctx, &svcdef.StringMsg{Str: input})
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	fmt.Printf("r: %v\n", r.Str)
}
