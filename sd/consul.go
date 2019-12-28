package sd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	uriRegister   = "v1/agent/service/register"
	uriUnregister = "v1/agent/service/deregister"
)

// ConsulRegisteror implements service registeror backed by Consul
type ConsulRegisteror struct {
	httpClient *http.Client
	addr       string
}

// NewConsulRegisteror creates new ConsulRegisteror
func NewConsulRegisteror(addr string) *ConsulRegisteror {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	reg := &ConsulRegisteror{
		addr:       addr,
		httpClient: httpClient,
	}

	return reg
}

// Register implements method of SvcRegisteror
func (r ConsulRegisteror) Register(svc *SvcDef) (interface{}, error) {
	regurl := fmt.Sprintf("%s/%s", r.addr, uriRegister)

	payload := map[string]interface{}{
		"name":    svc.Name,
		"id":      svc.ID,
		"address": svc.Addr,
		"port":    svc.Port,
	}

	if svc.HealthCheck != nil {
		check := map[string]interface{}{}
		hasErr := false
		switch svc.HealthCheck.Type {
		case "http":
			check["http"] = svc.HealthCheck.Content
		case "script":
			check["script"] = svc.HealthCheck.Content
		default:
			logger.Warn("Unsupported health check:", svc.HealthCheck.Type)
			hasErr = true
		}

		if !hasErr {
			interval := 3
			if svc.HealthCheck.Interval > 0 {
				interval = svc.HealthCheck.Interval
			}
			timeout := 1
			if svc.HealthCheck.Timeout > 0 {
				timeout = svc.HealthCheck.Timeout
			}
			check["interval"] = fmt.Sprintf("%ds", interval)
			check["timeout"] = fmt.Sprintf("%ds", timeout)

			payload["check"] = check
		}
	}

	// get tags from tags & env
	tags := make([]string, 0)
	if svc.Tags != nil && len(svc.Tags) > 0 {
		tags = append(tags, svc.Tags...)
	}
	if nodeType, ok := os.LookupEnv("NODE_TYPE"); ok && nodeType != "" {
		tags = append(tags, nodeType)
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("[SD] Fail to build payload from url: %s, %v", regurl, err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPut, regurl, bytes.NewBuffer(jsonBytes))
	if err != nil {
		logger.Errorf("[SD] Fail to build request from url: %s, %v", regurl, err)
		return nil, err
	}

	logger.Infof("[SD] Payload: %s", string(jsonBytes))
	resp, err := r.httpClient.Do(req)
	if err != nil {
		logger.Errorf("[SD] Fail to PUT request url: %s, %v", regurl, err)
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	logger.Infof("[SD] service(%s id=%s, addr=%s:%d) registerd, return:[%s]", svc.Name, svc.ID, svc.Addr, svc.Port, string(body))
	return true, nil
}

// Unregister implements method of SvcRegisteror
func (r ConsulRegisteror) Unregister(svc *SvcDef) (interface{}, error) {
	regurl := fmt.Sprintf("%s/%s/%s", r.addr, uriUnregister, svc.ID)

	req, err := http.NewRequest(http.MethodPut, regurl, nil)
	if err != nil {
		logger.Errorf("[SD] Fail to build request from url: %s, %v", regurl, err)
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		logger.Errorf("[SD] Fail to PUT request url: %s, %v", regurl, err)
		return nil, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	logger.Infof("[SD] service(%s id=%s) unregisterd, return:[%s]", svc.Name, svc.ID, string(body))
	return true, nil
}
