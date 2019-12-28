package lb

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/go-kit/kit/endpoint"
)

// LoadBalancer defines how to select endpoint from a list of endpoints with tag labels
type LoadBalancer interface {
	Select(uri, method string, endpoints map[string]endpoint.Endpoint, tagMap map[string][]string) (endpoint.Endpoint, string, error)
}

type randomLoadBalancer struct {
}

// NewRandomLoadBalancer creates an weighted random LB
func NewRandomLoadBalancer() LoadBalancer {
	lb := &randomLoadBalancer{}
	return lb
}

var count = 0
var stg = 0

func (r randomLoadBalancer) Select(uri, method string, endpoints map[string]endpoint.Endpoint, tagMap map[string][]string) (endpoint.Endpoint, string, error) {
	keys := make([]string, 0)
	steps := make([]int, 0)
	totalWeight := 0

	// build keys and weights
	for addr := range endpoints {
		keys = append(keys, addr)

		weight := 100
		if tags, ok := tagMap[addr]; ok {
			for _, tag := range tags {
				if tag == "stg" {
					weight = 1
				} else if strings.Index(tag, "weight_") == 0 {
					arr := strings.Split(tag, "_")
					if len(arr) == 2 {
						wStr := arr[1]
						if w, err := strconv.Atoi(wStr); err == nil {
							if w < 0 {
								w = 0
							} else if w > 100 {
								w = 100
							}
							weight = w
						}
					}
				}
			}
		}
		totalWeight += weight
		steps = append(steps, totalWeight)
	}

	if len(keys) == 0 {
		err := fmt.Errorf("no endpoint for %s-%s, all nodes dead", method, uri)
		return nil, "", err
	}

	if len(keys) == 1 {
		addr := keys[0]
		ep := endpoints[addr]
		return ep, addr, nil
	}

	num := rand.Intn(totalWeight)
	idx := 0
	for i, step := range steps {
		if num < step {
			idx = i
			break
		}
	}

	addr := keys[idx]
	ep := endpoints[addr]
	return ep, addr, nil
}
