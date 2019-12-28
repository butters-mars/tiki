package sd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/butters-mars/tiki/utils"
)

const (
	// DefaultConsulAddr defaut consul addr
	DefaultConsulAddr = "http://127.0.0.1:8500"
	// DefaultCheckEndpoint default http check endpont
	DefaultCheckEndpoint = "/healthcheck"
)

// InitServiceDiscovery init SD
func InitServiceDiscovery(sdConfig *ServiceDiscoverySt, listenAddr string) (register SvcRegisteror, svc *SvcDef, err error) {
	if sdConfig.Type != "consul" {
		err = fmt.Errorf("service discovery type [%s] not supported", sdConfig.Type)
		return
	}

	consulAddr := DefaultConsulAddr
	if sdConfig.RegAddr != "" {
		consulAddr = sdConfig.RegAddr
	}

	logger.Info("[SD] using consul service discovery")
	cReg := NewConsulRegisteror(consulAddr)
	ip := utils.GetIP()

	if ip == "" {
		err = fmt.Errorf("cannot get IP")
		return
	}

	if sdConfig.SvcName == "" {
		err = fmt.Errorf("service name not given")
		return
	}

	var port int
	segs := strings.Split(listenAddr, ":")
	if len(segs) <= 1 {
		port = 80
	} else {
		port, err = strconv.Atoi(segs[1])
		if err != nil {
			err = fmt.Errorf("cannot get port from server->listen_addr: %s", listenAddr)
			return
		}
	}

	hcEndpoint := sdConfig.CheckEndpoint
	if hcEndpoint == "" {
		hcEndpoint = DefaultCheckEndpoint
	} else if hcEndpoint[0] != '/' {
		hcEndpoint = "/" + hcEndpoint
	}

	hcAddr := sdConfig.CheckAddr
	if hcAddr == "" {
		hcAddr = fmt.Sprintf("%s:%d", ip, port)
	}

	svcName := sdConfig.SvcName
	id := fmt.Sprintf("%s-%d-%s", ip, port, strings.Replace(svcName, ".", "_", -1))
	svc = &SvcDef{
		ID:   id,
		Name: sdConfig.SvcName,
		Addr: ip,
		Port: port,
		HealthCheck: &SvcHealthChk{
			Type:    "http",
			Content: fmt.Sprintf("http://%s%s", hcAddr, hcEndpoint),
		},
	}

	logger.Infof("[SD] registering service[%s id=%s addr=(%s:%d)] to %s ...",
		sdConfig.SvcName, id, ip, port, consulAddr)
	_, err = cReg.Register(svc)
	if err != nil {
		return
	}

	reg := SvcRegisteror(*cReg)
	register = reg

	return
}
