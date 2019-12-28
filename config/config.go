package config

import (
	consulapi "github.com/hashicorp/consul/api"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// Config defines all configuration of an application
type Config struct {
	APPName          string                   `yaml:"appname"`
	Port             int                      `yaml:"port"`
	Tracing          *jaegercfg.Configuration `yaml:"tracing"`
	ServiceDiscovery ServiceDiscoveryCfg      `yaml:"service-discovery"`
	Auth             *AuthConfig              `yaml:"auth"`
	UpstreamSetting  string                   `yaml:"upstream-setting"`
	Properties       map[string]string        `yaml:"props"`
}

// ServiceDiscoveryCfg provides config of service discovery
type ServiceDiscoveryCfg struct {
	Type   string            `yaml:"type"` // consul or direct, default is direct
	Consul *consulapi.Config `yaml:"consul"`
}

// AuthConfig provides auth configuration
type AuthConfig struct {
	TLS      bool   `yaml:"tls"`
	CertFile string `yaml:"cert"`
	KeyFile  string `yaml:"key"`
}

// ServiceDiscoverySt defines config for consul discovery
// type ServiceDiscoverySt struct {
// 	Type          string `json:"type" yaml:"type"`
// 	RegAddr       string `json:"regaddr" yaml:"regaddr"`
// 	SvcName       string `json:"svcname" yaml:"svcname"`
// 	CheckEndpoint string `json:"check" yaml:"check"`
// 	CheckAddr     string `json:"checkaddr" yaml:"checkaddr"`
// 	DiscoveryInfo string `json:"discinfo" yaml:"discinfo"`
// }
