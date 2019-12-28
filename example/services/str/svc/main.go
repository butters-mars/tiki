package main

import (
	"flag"

	"github.com/butters-mars/tiki/auth"

	"google.golang.org/grpc"
	"github.com/butters-mars/tiki/app"
	"github.com/butters-mars/tiki/example/services/str/impl"
	"github.com/butters-mars/tiki/example/svcdef"
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
