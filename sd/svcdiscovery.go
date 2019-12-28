package sd

import "github.com/butters-mars/tiki/logging"

var logger = logging.Logger

// ServiceDiscoverySt defines config for consul discovery
type ServiceDiscoverySt struct {
	Type          string `json:"type" yaml:"type"`
	RegAddr       string `json:"regaddr" yaml:"regaddr"`
	SvcName       string `json:"svcname" yaml:"svcname"`
	CheckEndpoint string `json:"check" yaml:"check"`
	CheckAddr     string `json:"checkaddr" yaml:"checkaddr"`
	DiscoveryInfo string `json:"discinfo" yaml:"discinfo"`
}

// SvcHealthChk defines health check of a service
type SvcHealthChk struct {
	Type     string
	Content  string
	Interval int
	Timeout  int
}

// SvcDef defines a service to register
type SvcDef struct {
	Name        string
	ID          string
	Addr        string
	Port        int
	Tags        []string
	HealthCheck *SvcHealthChk
}

// SvcRegisteror represents an interface for Service Discovery registration
type SvcRegisteror interface {
	Register(svc *SvcDef) (interface{}, error)
	Unregister(svc *SvcDef) (interface{}, error)
}
