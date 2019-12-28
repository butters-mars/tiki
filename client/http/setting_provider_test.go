package http

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

type ES struct {
	URI    string `yaml:"uri"`
	Method string `yaml:"method"`
	//lbType   string
	//retry    *Retry
	CBConfig hystrix.CommandConfig `yaml:"hystrix"`
}

type XXX struct {
	Settings []ES
}

func TestSettingLoad(t *testing.T) {
	var str string

	str = `
settings:
 test.srv.ns:
  - uri: /hello/jack/134
    method: POST
    hystrix:
     timeout: 2000
     max_concurrent_requests: 20
`

	fn := fmt.Sprintf("/tmp/%d", time.Now().Nanosecond())
	err := ioutil.WriteFile(fn, []byte(str), 0644)
	if err != nil {
		t.Errorf("fail to write file: %v", err)
		return
	}

	p, err := NewFileSettingProvider(fn)
	if err != nil {
		t.Errorf("fail to create provider %v", err)
		return
	}

	s, err := p.GetSettings("test.srv.ns")
	if err != nil {
		t.Errorf("fail to load : %v", err)
		return
	}

	if st, ok := s["/hello/jack/134-POST"]; ok {
		if st.URI != "/hello/jack/134" || st.Method != "POST" || st.CBConfig.Timeout != 2000 {
			t.Errorf("wrong parse result")
		}

		return
	}

}

// Timeout                int `json:"timeout"`
// MaxConcurrentRequests  int `json:"max_concurrent_requests"`
// RequestVolumeThreshold int `json:"request_volume_threshold"`
// SleepWindow            int `json:"sleep_window"`
// ErrorPercentThreshold  int `json:"error_percent_threshold"`
