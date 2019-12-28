package http

import (
	"context"
	"strings"
	"testing"

	"github.com/afex/hystrix-go/hystrix"
)

var (
	s1 = EndpointSetting{
		URI:    "/good",
		Method: "GET",
		CBConfig: hystrix.CommandConfig{
			Timeout: 100,
		},
	}

	s2 = EndpointSetting{
		URI:    "/1ms",
		Method: "GET",
		CBConfig: hystrix.CommandConfig{
			Timeout: 1,
		},
	}
	s3 = EndpointSetting{
		URI:    "/5ms-15ms",
		Method: "GET",
		CBConfig: hystrix.CommandConfig{
			Timeout:                5,
			RequestVolumeThreshold: 5,
		},
	}
)

type mockSettingProvider struct {
}

func (p mockSettingProvider) GetSettings(tgt string) (map[string]EndpointSetting, error) {
	if tgt == "test" {
		return map[string]EndpointSetting{
			"/good-GET":     s1,
			"/1ms-GET":      s2,
			"/5ms-15ms-GET": s3,
		}, nil
	}

	return nil, nil
}

func (p mockSettingProvider) SetHandler(h func(EndpointSetting) error) {
}

func TestApiClient(t *testing.T) {
	sdConfig := "consul::localhost:8500/dc1"

	SetSource("A")
	SetServiceDiscoveryCfg(sdConfig)

	SetSettingProvider(mockSettingProvider{})

	//cl := NewDefaultClient("localhost:8886", []*EndpointSetting{s1, s2})
	cl := GetDefaultClientWithSD("test", true)

	resp := make(map[string]interface{})
	err := cl.Do(context.TODO(), "/good", "GET", nil, &resp)
	if err != nil {
		t.Errorf("fail to call: %v", err)
		return
	}

	err = cl.Do(context.TODO(), "/badjson", "GET", nil, &resp)
	if err == nil {
		t.Errorf("should fail to call badjson: %v", resp)
		return
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("should be unmarshal error: %v", err)
		return
	}

	err = cl.Do(context.TODO(), "/1ms", "GET", nil, &resp)
	if err == nil {
		t.Errorf("should fail to call 1ms")
		return
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("should be timeout error")
		return
	}

	err = cl.Do(context.TODO(), "/path_not_exist", "GET", nil, &resp)
	if err == nil {
		t.Errorf("should fail to call path_not_exist")
		return
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("should be 404 error")
		return
	}

	cl = GetDefaultClientWithSD("badxxsdssfsfs", true)
	resp = make(map[string]interface{})
	err = cl.Do(context.TODO(), "/haha", "GET", nil, &resp)
	if err == nil {
		t.Errorf("should fail to call bad address")
		return
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("should be no such host error")
		return
	}

	cl = GetDefaultClientWithSD("test", true)
	resp = make(map[string]interface{})
	for i := 0; i < 5; i++ {
		err = cl.Do(context.TODO(), "/5ms-15ms", "GET", nil, &resp)
	}
	err = cl.Do(context.TODO(), "/5ms-15ms", "GET", nil, &resp)
	if err == nil {
		t.Errorf("circuitbreaker should already be open")
		return
	}
	if !strings.Contains(err.Error(), "circuit open") {
		t.Errorf("should be circuit open error")
		return
	}
}
