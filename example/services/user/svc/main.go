package main

import (
	"context"
	"flag"

	"google.golang.org/grpc"
	"github.com/butters-mars/tiki/app"
	"github.com/butters-mars/tiki/example/svcdef"
)

// App the app instance
var (
	App     *app.App
	cfgName = flag.String("config", "config", "config file name")
)

func main() {
	App = app.New(*cfgName)
	App.RegisterGRPCServer(func(s *grpc.Server) {
		svcdef.RegisterUserServer(s, &srv{})
	})

	App.Start()
}

type srv struct{}

func (s srv) GetProfile(ctx context.Context, req *svcdef.GetProfileReq) (*svcdef.Profile, error) {
	return &svcdef.Profile{
		Id:   req.Id,
		Name: "jack",
		Avatar: &svcdef.Avatar{
			Url: "http://avatar.com/jack_smith",
		},
		Contact: &svcdef.Contact{
			Email: "jack@google.com",
			Phone: "+861129109481",
		},
		Addresses: []*svcdef.Address{
			&svcdef.Address{
				Province: "Beijing",
				City:     "Beijing",
			},
			&svcdef.Address{
				Province: "Hainan",
				City:     "Sanya",
			},
		},
	}, nil
}
