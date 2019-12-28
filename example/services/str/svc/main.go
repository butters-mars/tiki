package main

import (
	"flag"

	"github.com/tiki/auth"

	"google.golang.org/grpc"
	"github.com/tiki/app"
	"github.com/tiki/example/services/str/impl"
	"github.com/tiki/example/svcdef"
)

// App the app instance
var (
	App     *app.App
	cfgName = flag.String("config", "config", "config file name")
)

func main() {
	App = app.New(*cfgName)
	App.AuthFunc = auth.TokenAuth
	App.RegisterGRPCServer(func(s *grpc.Server) {
		svcdef.RegisterStringServer(s, &impl.StrSrv{})
	})

	App.Start()
}
