package lb

import (
	"context"
	"strings"
	"testing"

	"github.com/go-kit/kit/endpoint"
)

func TestRandomSelect(t *testing.T) {
	lb := NewRandomLoadBalancer()

	ep1 := func(context.Context, interface{}) (interface{}, error) { return 1, nil }
	ep2 := func(context.Context, interface{}) (interface{}, error) { return 2, nil }

	m1 := make(map[string]endpoint.Endpoint)

	_, _, err := lb.Select("", "", m1, nil)
	if err == nil {
		t.Error("should fail to select from empty")
		return
	}

	if strings.Index(err.Error(), "empty") == -1 {
		t.Errorf("error should contains empty: %v", err)
		return
	}

	m1["a"] = ep1
	ep, addr, err := lb.Select("", "", m1, nil)
	if err != nil {
		t.Errorf("should be ok to select from 1 ep: %v", err)
		return
	}
	if v, _ := ep(nil, nil); v != 1 || addr != "a" {
		t.Error("should select from ep1 a")
		return
	}

	m1["b"] = ep2
	tags := make(map[string][]string)
	tags["b"] = []string{"weight_25"}

	count := 0
	for i := 0; i < 100; i++ {
		_, addr, err := lb.Select("", "", m1, tags)
		if err != nil {
			t.Errorf("should be ok to select from 2: %v", err)
			return
		}

		if addr == "b" {
			count++
		}
	}

	if count > 24 || count < 15 {
		t.Errorf("b should be selected 15 - 24 : %d", count)
	}

	tags["b"] = []string{"stg"}
	count = 0
	for i := 0; i < 1000; i++ {
		_, addr, err := lb.Select("", "", m1, tags)
		if err != nil {
			t.Errorf("should be ok to select from 2: %v", err)
			return
		}

		if addr == "b" {
			count++
		}
	}

	if count < 5 || count > 20 {
		t.Errorf("b should be selected 5 - 20 : %d", count)
	}
}
