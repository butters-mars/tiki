package endpointer

import (
	"fmt"

	consul "github.com/hashicorp/consul/api"
	"github.com/tiki/client/sd/instancer"
	"github.com/tiki/logging"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	csd "github.com/go-kit/kit/sd/consul"
)

var logger = logging.Logger

// SDType defines service discovery types
type SDType string

const (
	// SDTypeConsul consul
	SDTypeConsul SDType = "consul"
	// SDTypeNone no service discovery
	SDTypeNone SDType = "none"
)

// WithTag extends Endpointer with tag support
type WithTag interface {
	sd.Endpointer
	GetTagMap() map[string][]string
}

type consulEndpointer struct {
	consulClient *consul.Client
	sdClient     csd.Client
	instancer    *instancer.Instancer
	endpointer   sd.Endpointer
}

type sdLogger struct {
}

func (l sdLogger) Log(keyvals ...interface{}) error {
	logger.Info("[Endpointer] %v", keyvals)
	return nil
}

var options []sd.EndpointerOption

// NewConsulEndpointer creates an endpointer backed by consul service discovery
func NewConsulEndpointer(sdCfgMap map[string]string, sdFactory sd.Factory, service string, tags []string, passingOnly bool) (epr WithTag, err error) {
	if sdCfgMap == nil {
		err = fmt.Errorf("empty cfg map")
		return
	}

	if _type, ok := sdCfgMap["type"]; !ok || _type != "consul" {
		err = fmt.Errorf("only consul sd supperted right now")
		return
	}

	config := &consul.Config{
		Address:    sdCfgMap["address"],
		Datacenter: sdCfgMap["datacenter"],
	}

	consulClient, err := consul.NewClient(config)
	if err != nil {
		return
	}

	sdClient := csd.NewClient(consulClient)
	// use local version of instancer to provide tagging support
	instancer := instancer.NewInstancer(sdClient, sdLogger{}, service, tags, passingOnly, nil)
	endpointer := sd.NewEndpointer(instancer, sdFactory, sdLogger{}, options...)

	epr = &consulEndpointer{
		consulClient: consulClient,
		sdClient:     sdClient,
		instancer:    instancer,
		endpointer:   endpointer,
	}

	return
}

func (r consulEndpointer) Endpoints() (eps []endpoint.Endpoint, err error) {
	eps, err = r.endpointer.Endpoints()
	return
}

func (r consulEndpointer) GetTagMap() (tagMap map[string][]string) {
	return r.instancer.GetTagMap()
}

type fixedEndpointer struct {
	instancer  sd.Instancer
	endpointer sd.Endpointer
}

func (f fixedEndpointer) GetTagMap() (tagMap map[string][]string) { return }
func (f fixedEndpointer) Endpoints() (eps []endpoint.Endpoint, err error) {
	eps, err = f.endpointer.Endpoints()
	return
}

// NewDirectEndpointer creates an endpointer with no service discovery, and connect given host directly
func NewDirectEndpointer(host string, sdFactory sd.Factory) (endpointer WithTag, err error) {
	instancer := sd.FixedInstancer([]string{host})
	epr := sd.NewEndpointer(instancer, sdFactory, sdLogger{})

	endpointer = &fixedEndpointer{
		instancer:  instancer,
		endpointer: epr,
	}

	return
}
