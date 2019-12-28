package instancer

import (
	"fmt"
	"io"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	csd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/util/conn"
	consul "github.com/hashicorp/consul/api"

	"github.com/butters-mars/tiki/logging"
)

const defaultIndex = 0

var logger = logging.Logger

// Instancer yields instances for a service in Consul.
type Instancer struct {
	cache       *Cache
	client      csd.Client
	logger      log.Logger
	service     string
	tags        []string
	passingOnly bool
	quitc       chan struct{}

	listener Listener

	// NOTE
	// add these to support tagging instances with "prod", "stg", etc.
	tagMap map[string][]string
}

// Listener handles update events
type Listener func(instances []string, tags map[string][]string, err error)

// NewInstancer returns a Consul instancer that publishes instances for the
// requested service. It only returns instances for which all of the passed tags
// are present.
func NewInstancer(client csd.Client, logger log.Logger, service string, tags []string, passingOnly bool, listener Listener) *Instancer {
	s := &Instancer{
		cache:       NewCache(),
		client:      client,
		logger:      log.With(logger, "service", service, "tags", fmt.Sprint(tags)),
		service:     service,
		tags:        tags,
		passingOnly: passingOnly,
		quitc:       make(chan struct{}),
		tagMap:      make(map[string][]string),
		listener:    listener,
	}

	instances, index, err := s.getInstances(defaultIndex, nil)
	if err == nil {
		s.logger.Log("instances", len(instances))
	} else {
		s.logger.Log("err", err)
	}

	s.cache.Update(sd.Event{Instances: instances, Err: err})
	if s.listener != nil {
		s.listener(instances, s.tagMap, err)
	}
	go s.loop(index)
	return s
}

// SetListener set the update listener
func (s *Instancer) SetListener(l Listener) {
	s.listener = l
}

// Stop terminates the instancer.
func (s *Instancer) Stop() {
	close(s.quitc)
}

func (s *Instancer) loop(lastIndex uint64) {
	var (
		instances []string
		err       error
		d         = 10 * time.Millisecond
	)
	for {
		instances, lastIndex, err = s.getInstances(lastIndex, s.quitc)
		switch {
		case err == io.EOF:
			return // stopped via quitc
		case err != nil:
			s.logger.Log("err", err)
			time.Sleep(d)
			d *= 2
			s.cache.Update(sd.Event{Err: err})
			if s.listener != nil {
				s.listener(instances, s.tagMap, err)
			}
		default:
			s.cache.Update(sd.Event{Instances: instances})
			d = 10 * time.Millisecond
			if s.listener != nil {
				s.listener(instances, s.tagMap, nil)
			}
		}
	}
}

func (s *Instancer) getInstances(lastIndex uint64, interruptc chan struct{}) ([]string, uint64, error) {
	tag := ""
	if len(s.tags) > 0 {
		tag = s.tags[0]
	}

	// Consul doesn't support more than one tag in its service query method.
	// https://github.com/hashicorp/consul/issues/294
	// Hashi suggest prepared queries, but they don't support blocking.
	// https://www.consul.io/docs/agent/http/query.html#execute
	// If we want blocking for efficiency, we must filter tags manually.

	type response struct {
		instances []string
		index     uint64
	}

	var (
		errc = make(chan error, 1)
		resc = make(chan response, 1)
	)

	go func() {
		entries, meta, err := s.client.Service(s.service, tag, s.passingOnly, &consul.QueryOptions{
			WaitIndex: lastIndex,
		})
		if err != nil {
			errc <- err
			return
		}
		if len(s.tags) > 1 {
			entries = filterEntries(entries, s.tags[1:]...)
		}

		// set tags
		tagMap := make(map[string][]string)
		instances := makeInstances(entries)
		for i, entry := range entries {
			tags := make([]string, 0)
			if entry.Service != nil && entry.Service.Tags != nil {
				for _, tag := range entry.Service.Tags {
					tags = append(tags, tag)
				}
			}

			tagMap[instances[i]] = tags
		}
		s.tagMap = tagMap
		logger.Infof("[Instancer] update tag map of %s: %v", s.service, tagMap)

		resc <- response{
			instances: instances,
			index:     meta.LastIndex,
		}
	}()

	select {
	case err := <-errc:
		return nil, 0, err
	case res := <-resc:
		return res.instances, res.index, nil
	case <-interruptc:
		return nil, 0, io.EOF
	}
}

// Register implements Instancer.
func (s *Instancer) Register(ch chan<- sd.Event) {
	s.cache.Register(ch)
}

// Deregister implements Instancer.
func (s *Instancer) Deregister(ch chan<- sd.Event) {
	s.cache.Deregister(ch)
}

// GetTagMap returns tags as map
func (s *Instancer) GetTagMap() map[string][]string {
	return s.tagMap
}

func filterEntries(entries []*consul.ServiceEntry, tags ...string) []*consul.ServiceEntry {
	var es []*consul.ServiceEntry

ENTRIES:
	for _, entry := range entries {
		ts := make(map[string]struct{}, len(entry.Service.Tags))
		for _, tag := range entry.Service.Tags {
			ts[tag] = struct{}{}
		}

		for _, tag := range tags {
			if _, ok := ts[tag]; !ok {
				continue ENTRIES
			}
		}
		es = append(es, entry)
	}

	return es
}

func makeInstances(entries []*consul.ServiceEntry) []string {
	instances := make([]string, len(entries))
	for i, entry := range entries {
		addr := entry.Node.Address
		if entry.Service.Address != "" {
			addr = entry.Service.Address
		}
		instances[i] = fmt.Sprintf("%s:%d", addr, entry.Service.Port)
	}
	return instances
}
