package http

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/butters-mars/tiki/client/http/middleware"
	"github.com/butters-mars/tiki/client/sd/endpointer"
	"github.com/butters-mars/tiki/logging"
)

var (
	source = ""

	serviceDiscoveryCfgStr = "consul::localhost:8500/dc1" // consul, localhost, dc1
	serviceDiscoveryCfg    = parseSDCfg(serviceDiscoveryCfgStr)

	settingProvider SettingProvider

	logger = logging.L

	//mutext = sync.RWMutex{}
	// use global map to avoid recreating apiclient
	//globalClients = make(map[string]*DefaultClient)
)

// Client the new client that supports circuitbreak, client-side lb, metrics etc.
type Client interface {
	Do(ctx context.Context, uri, method string, param interface{}, resp interface{}) (err error)
	DoRaw(ctx context.Context, uri, method string, param interface{}) (resp []byte, code int, err error)
}

// DefaultClient provides a default implementation of ApiClient
type DefaultClient struct {
	id                  string
	host                string
	settings            map[string]EndpointSetting
	endpoints           map[string]*endpointClient
	mutex               *sync.RWMutex
	useServiceDiscovery bool
}

// SetSource set the src tag for metrics
func setSource(s string) {
	source = s
}

// SetServiceDiscoveryCfg setup service discovery configuration
func SetServiceDiscoveryCfg(cfg string) {
	serviceDiscoveryCfgStr = cfg
	serviceDiscoveryCfg = parseSDCfg(cfg)
}

// SetSettingProvider setup endpoint setting provider
func SetSettingProvider(p SettingProvider) {
	settingProvider = p
}

// SetupClient initializes global api client setting
func SetupClient(source, upstreamSetting, discoveryInfo string) {
	setSource(source)
	initMetrics()
	if upstreamSetting != "" {
		provider, err := NewFileSettingProvider(upstreamSetting)
		if err != nil {
			logger.Errorf("fail to create setting provider from %s: %v", upstreamSetting, err)
		} else {
			logger.Infof("upstream setting loaded from %s", upstreamSetting)
			SetSettingProvider(provider)
		}
	}

	if discoveryInfo != "" {
		SetServiceDiscoveryCfg(discoveryInfo)
	}
}

// GetDefaultClient gets an ApiClient for a given host
// func GetDefaultClient(host string) *DefaultClient {
// 	return GetDefaultClientWithSD(host, true)
// }

// GetDefaultClientWithSD gets an ApiClient for a given host and specifying whether using consul service discovery
// func GetDefaultClientWithSD(host string, useServiceDiscovery bool) *DefaultClient {
// 	mutext.RLock()
// 	if client, ok := globalClients[host]; ok {
// 		mutext.RUnlock()
// 		return client
// 	}
// 	mutext.RUnlock()

// 	// use global map to avoid recreating apiclient for every quest
// 	// see @xclib/common/httpsvr/context.go Next() for detail
// 	mutext.Lock()
// 	defer mutext.Unlock()

// 	if client, ok := globalClients[host]; ok {
// 		return client
// 	}

// NewClient returns client with given host, and uses sd by default
func NewClient(host string) Client {
	return NewClientWithSD(host, true)
}

// NewClientWithSD returns client with given host
func NewClientWithSD(host string, useServiceDiscovery bool) Client {
	id := fmt.Sprintf("%s-%d-%d", host, time.Now().Nanosecond(), rand.Intn(10000))
	logger.Info("creating new api client %s", id)

	var settings map[string]EndpointSetting
	if settingProvider != nil {
		_settings, err := settingProvider.GetSettings(host)
		if err != nil {
			logger.Error("fail to get setting from settingProvider: %v", err)
		}
		settings = _settings
	}

	client := DefaultClient{
		id:                  id,
		host:                host,
		settings:            settings,
		endpoints:           make(map[string]*endpointClient),
		mutex:               &sync.RWMutex{},
		useServiceDiscovery: useServiceDiscovery,
	}

	for key, setting := range settings {
		c, err := client.createEndpointClient(&setting)
		if err != nil {
			logger.Error("fail to create endpoint client for %s, err: %v", key, err)
			continue
		}
		client.endpoints[key] = c
	}

	return client
}

// SetEndpointSetting dynamically changes setting (NOT IMPL YET)
func (c DefaultClient) SetEndpointSetting(setting *EndpointSetting) (err error) {
	return
}

// SetEndpointSettings dynamically changes setting (NOT IMPL YET)
func (c DefaultClient) SetEndpointSettings(settings []*EndpointSetting) (err error) {
	return
}

// InitMetrics init metrics
func initMetrics() {
	middleware.InitMetrics()
}

// Do delegates the request to endpoint client
func (c DefaultClient) Do(ctx context.Context, uri, method string, param interface{}, resp interface{}) (err error) {
	client, err := c.getEndpointClient(uri, method)
	if err != nil {
		logger.Error("cannot get endpoint client for %s-%s, err: %v", uri, method, err)
		return
	}

	if client == nil {
		logger.Error("nil client for %s_%s", uri, method)
		err = fmt.Errorf("nil client")
		return
	}

	return client.Do(ctx, uri, method, param, resp)
}

// DoRaw delegates the request to endpoint client, supports byte[] as param, and returns []byte, status code
func (c DefaultClient) DoRaw(ctx context.Context, uri, method string, param interface{}) (resp []byte, code int, err error) {
	client, err := c.getEndpointClient(uri, method)
	if err != nil {
		logger.Error("cannot get endpoint client for %s-%s, err: %v", uri, method, err)
		return
	}

	if client == nil {
		logger.Error("nil client for %s_%s", uri, method)
		err = fmt.Errorf("nil client")
		return
	}

	return client.DoRaw(ctx, uri, method, param)
}

func (c DefaultClient) getEndpointClient(uri, method string) (*endpointClient, error) {
	key := fmt.Sprintf("%s-%s", method, uri)

	c.mutex.RLock()
	if client, ok := c.endpoints[key]; ok {
		c.mutex.RUnlock()
		return client, nil
	}
	c.mutex.RUnlock()

	client, err := c.createByDefaultCfg(key, uri, method)
	keys := []string{}
	for _key := range c.endpoints {
		keys = append(keys, _key)
	}

	return client, err
}

func (c DefaultClient) createByDefaultCfg(key, uri, method string) (*endpointClient, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var client *endpointClient
	if _, ok := c.endpoints[key]; ok {
		client = c.endpoints[key]
		return client, nil
	}

	setting := &EndpointSetting{
		URI:    uri,
		Method: method,
		//lbType:   "",
		//retry:    nil,
		CBConfig: middleware.DefaultCBConfig,
	}
	_c, err := c.createEndpointClient(setting)
	if err != nil {
		logger.Error("[%s]cannot create endpoint client for %s_%s", c.id, uri, method)
		return nil, err
	}

	if _c == nil {
		logger.Error("[%s]nil endpoint client for %s_%s", c.id, uri, method)
		return nil, fmt.Errorf("nil endpoint client")
	}

	c.endpoints[key] = _c
	client = _c

	return client, nil
}

func (c DefaultClient) createEndpointClient(setting *EndpointSetting) (*endpointClient, error) {
	logger.Info("[APIClient] %s create apiclient for %s%s-%s", c.id, c.host, setting.URI, setting.Method)

	defaultCBConfig := middleware.DefaultCBConfig

	if setting.CBConfig.Timeout == 0 {
		setting.CBConfig.Timeout = defaultCBConfig.Timeout
	}
	if setting.CBConfig.ErrorPercentThreshold == 0 {
		setting.CBConfig.ErrorPercentThreshold = defaultCBConfig.ErrorPercentThreshold
	}
	if setting.CBConfig.MaxConcurrentRequests == 0 {
		setting.CBConfig.MaxConcurrentRequests = defaultCBConfig.MaxConcurrentRequests
	}
	if setting.CBConfig.RequestVolumeThreshold == 0 {
		setting.CBConfig.RequestVolumeThreshold = defaultCBConfig.RequestVolumeThreshold
	}
	if setting.CBConfig.SleepWindow == 0 {
		setting.CBConfig.SleepWindow = defaultCBConfig.SleepWindow
	}

	sdType := endpointer.SDTypeNone
	if c.useServiceDiscovery {
		sdType = endpointer.SDTypeConsul
	}
	return newEndpointClient(c.host, setting, sdType)
}

func parseSDCfg(cfg string) map[string]string {
	var cfgMap map[string]string
	if cfg == "" {
		logger.Error("empty service discovery config")
		return nil
	}

	segs := strings.Split(cfg, "::")
	if len(segs) < 2 {
		logger.Error("bad service discovery config: %s", cfg)
		return nil
	}

	_type := segs[0]
	info := segs[1]
	switch _type {
	case "consul":
		segs = strings.Split(info, "/")
		if len(segs) != 2 {
			logger.Error("bad consul service discovery config: %s", cfg)
			return nil
		}
		cfgMap = make(map[string]string)
		cfgMap["type"] = _type
		cfgMap["address"] = segs[0]
		cfgMap["datacenter"] = segs[1]
		return cfgMap

	default:

	}

	logger.Error("unsupported service discovery config: %s", cfg)
	return nil
}
