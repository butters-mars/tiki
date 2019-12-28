package main

import (
	"flag"

	"github.com/butters-mars/tiki/app"
)

var (
	cfgName = flag.String("config", "config", "config file name")
	// App stands for a grpc app instance
	App = app.New(*cfgName)
)

func main() {}
