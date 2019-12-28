package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"

	"github.com/tiki/client/http/lb"
	"github.com/tiki/client/http/middleware"
	"github.com/tiki/client/sd/endpointer"
)

// endpointClient represents client for a certain (http://host/uri - METHOD) which contains several
// endpoints, and could be lb-ed by a LoadBalancer
type endpointClient struct {
	host    string
	uri     string
	method  string
	setting *EndpointSetting
	lb      lb.LoadBalancer

	httpClient *http.Client
	taggedEPR  endpointer.WithTag
	sdType     endpointer.SDType

	endpointMap map[string]endpoint.Endpoint
	mutext      *sync.RWMutex
}

type requestBuilder func(addr string, uri string, method string, body []byte) (*http.Request, error)

// Retry retry policies
type Retry struct {
}

// EndpointSetting hystrix & retry settings for endpoint
type EndpointSetting struct {
	URI      string                `yaml:"uri"`
	Method   string                `yaml:"method"`
	CBConfig hystrix.CommandConfig `yaml:"hystrix"`
	//lbType   string
	//retry    *Retry
}

func newEndpointClient(host string, setting *EndpointSetting, sdType endpointer.SDType) (ep *endpointClient, err error) {
	ep = &endpointClient{
		host:        host,
		uri:         setting.URI,
		method:      setting.Method,
		setting:     setting,
		endpointMap: make(map[string]endpoint.Endpoint),
		lb:          lb.NewRandomLoadBalancer(),
		sdType:      sdType,
		mutext:      &sync.RWMutex{},
	}

	err = ep.init()
	return
}

func (client *endpointClient) init() (err error) {
	// http client
	timeout := time.Duration(client.setting.CBConfig.Timeout) * time.Millisecond
	maxConcurrentRequests := client.setting.CBConfig.MaxConcurrentRequests
	client.httpClient = client.createHTTPClient(timeout, maxConcurrentRequests)

	// middleware
	source = normal(source)
	host := normal(client.host)
	uri := normal(client.uri)
	cmdName := fmt.Sprintf("%s-%s-%s-%s", source, host, uri, client.method)
	circuitbreaker := middleware.CircuitBreaker(cmdName, client.setting.CBConfig)
	metrics := middleware.Metrics(source, host, uri, client.method)
	tracing := middleware.Tracing(client.uri)
	middleware := endpoint.Chain(circuitbreaker, metrics, tracing, middleware.Cleanup())

	// sd resolver
	factory := client.createEndpointFactory(client.httpClient, middleware)
	var epr endpointer.WithTag
	if client.sdType == endpointer.SDTypeConsul {
		epr, err = endpointer.NewConsulEndpointer(serviceDiscoveryCfg, factory, client.host, nil, true)
	} else if client.sdType == endpointer.SDTypeNone {
		epr, err = endpointer.NewDirectEndpointer(client.host, factory)
	} else {
		err = fmt.Errorf("unsupported sdType :%v", client.sdType)
	}

	if err != nil {
		return
	}
	client.taggedEPR = epr
	return
}

func normal(str string) string {
	return strings.Replace(str, "-", "_", -1)
}

func (client *endpointClient) DoRaw(ctx context.Context, uri, method string, param interface{}) (resp []byte, code int, err error) {
	body := []byte("")

	// check if param is already []byte
	if bs, ok := param.([]byte); ok {
		body = bs
	} else if param != nil {
		var bs []byte
		bs, err = json.Marshal(param)
		if err != nil {
			logger.Error("json Marshal err: %v, param: %v", err, param)
			return
		}
		body = bs
	}

	_endpoint, addr, err := client.resolveHost(uri, method)
	if err != nil {
		logger.Error("resolve host [%s] err: %v", client.host, err)
		return
	}

	url := fmt.Sprintf("http://%s%s", addr, uri)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("fail to build request for %s[%s], err: %v", url, string(body), err)
		return
	}

	response, err := _endpoint(ctx, req)
	if err != nil {
		logger.Error("fail to call %s[%s], err: %v", url, string(body), err)
		return
	}

	arr, ok := response.([]interface{})
	if !ok {
		logger.Error("resp is not array of interface")
		err = fmt.Errorf("resp not []interface{}")
		return
	}

	resp, ok = arr[0].([]byte)
	if !ok {
		logger.Error("arr[0] not []byte")
		err = fmt.Errorf("arr[0] not []byte")
		return
	}
	code, ok = arr[1].(int)
	if !ok {
		logger.Error("arr[1] not int")
		err = fmt.Errorf("arr[1] not int")
		return
	}

	return
}

func (client *endpointClient) Do(ctx context.Context, uri, method string, param interface{}, resp interface{}) (err error) {
	contentBytes, _, err := client.DoRaw(ctx, uri, method, param)
	if err != nil {
		return
	}

	err = json.Unmarshal(contentBytes, resp)
	if err != nil {
		logger.Error("fail to parse response body of %s [%s]: %v", uri, string(contentBytes), err)
		return
	}

	return
}

func (client *endpointClient) getTagMap() map[string][]string {
	return client.taggedEPR.GetTagMap()
}

func (client *endpointClient) resolveHost(uri, method string) (ep endpoint.Endpoint, addr string, err error) {
	client.mutext.RLock()
	defer client.mutext.RUnlock()

	// filter out all circuit-open endpoints
	endpoints := make(map[string]endpoint.Endpoint)
	for addr, ep := range client.endpointMap {
		cbKey := fmt.Sprintf("%s-%s-%s", addr, uri, method)
		if cbOpen, ok := middleware.IsCircuitOpen(cbKey); ok && cbOpen {
			// give 10% chance to let go of circuit-opened endpoint
			if rand.Intn(10) != 1 {
				logger.Warn("[EP] circuit %s open=true, ignore", cbKey)
				continue
			} else {
				logger.Info("[EP] circuit %s open=true, let go of it", cbKey)
			}
		}

		endpoints[addr] = ep
	}

	return client.lb.Select(uri, method, endpoints, client.getTagMap())
}

func (client *endpointClient) createHTTPClient(timeout time.Duration, maxConcurrentRequests int) *http.Client {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   timeout / 2,
			KeepAlive: 3600 * time.Second,
		}).Dial,
		MaxIdleConnsPerHost: maxConcurrentRequests,
		MaxIdleConns:        maxConcurrentRequests,
		TLSHandshakeTimeout: timeout,
		TLSClientConfig:     tlsCfg,
	}

	c := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return c
}

func (client *endpointClient) createEndpoint(httpClient *http.Client) endpoint.Endpoint {
	ep := func(ctx context.Context, req interface{}) (resp interface{}, err error) {
		httpReq, _ := req.(*http.Request)

		response, err := httpClient.Do(httpReq)
		if err != nil {
			return
		}

		if response.Body == nil {
			logger.Error("resp body is empty for %s", httpReq.URL.String())
			err = fmt.Errorf("resp body empty")
			return
		}
		defer response.Body.Close()

		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			logger.Error("Failed to read response body of %s: %v", httpReq.URL.String(), err)
			return
		}

		return []interface{}{content, response.StatusCode, response.Header}, err
	}

	return ep
}

type fakeCloser struct {
	addr    string
	action  string
	onClose func()
}

func (c fakeCloser) Close() error {
	logger.Info("[EPFactory] close endpoint %s", c.addr)
	middleware.CleanupEndpoint(c.action)
	c.onClose()
	return nil
}

func (client *endpointClient) createEndpointFactory(httpClient *http.Client, middleware endpoint.Middleware) sd.Factory {
	return func(addr string) (endpoint.Endpoint, io.Closer, error) {
		client.mutext.Lock()
		defer client.mutext.Unlock()

		ep := client.createEndpoint(httpClient)
		ep = middleware(ep)

		client.endpointMap[addr] = ep
		logger.Info("[EPFactory] create endpoint %s%s(%s), list=%s", client.host, client.uri, addr, client.debugEndpointMap())

		key := fmt.Sprintf("%s-%s-%s", addr, client.uri, client.method)
		closer := fakeCloser{
			addr:   addr,
			action: key,
			onClose: func() {
				client.mutext.Lock()
				defer client.mutext.Unlock()

				delete(client.endpointMap, addr)
				logger.Info("[EPFactory] delete endpoint %s%s(%s), list=%s", client.host, client.uri, addr, client.debugEndpointMap())
			},
		}

		return ep, closer, nil
	}
}

func (client *endpointClient) debugEndpointMap() string {
	str := " "
	i := 0
	for key := range client.endpointMap {
		str += key + " "
		i++
	}

	str = "[" + str + "](len=" + strconv.Itoa(i) + ")"

	return str
}
