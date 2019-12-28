package main

import (
	"context"
	"flag"

	"google.golang.org/grpc"
	"github.com/tiki/app"
	"github.com/tiki/example/svcdef"
)

// App the app instance
var (
	App     *app.App
	cfgName = flag.String("config", "config", "config file name")
)

func main() {
	App = app.New(*cfgName)
	App.RegisterGRPCServer(func(s *grpc.Server) {
		svcdef.RegisterMathServer(s, &mathSrv{})
	})

	App.Start()
}

type mathSrv struct{}

func (s mathSrv) Do(ctx context.Context, args *svcdef.Args) (*svcdef.Result, error) {
	a, b := args.A, args.B
	var r int32
	switch args.Op {
	case svcdef.OpType_Add:
		r = a + b
	case svcdef.OpType_Sub:
		r = a - b
	case svcdef.OpType_Mul:
		r = a * b
	}

	return &svcdef.Result{
		V: r,
	}, nil
}
