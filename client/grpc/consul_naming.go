package grpc

import (
	"fmt"
	"strings"
	"sync"

	csd "github.com/go-kit/kit/sd/consul"
	consul "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/naming"

	"github.com/tiki/client/sd/instancer"
)

type sdLogger struct {
}

func (l sdLogger) Log(keyvals ...interface{}) error {
	logger.Infof("[Naming] %v", keyvals)
	return nil
}

type consulResolver struct {
	naming.Resolver
	consulCfg *consul.Config
}

func newConsulResolver(cfg *consul.Config) naming.Resolver {
	return &consulResolver{
		consulCfg: cfg,
	}
}

func (c *consulResolver) Resolve(target string) (naming.Watcher, error) {
	return newConsulWatcher(c.consulCfg, target)
}

type updateMsg struct {
	instances []string
	tagMap    map[string][]string
	err       error
}

type consulWatcher struct {
	naming.Watcher
	instancer *instancer.Instancer
	entries   map[string]bool
	updateC   chan *updateMsg
	mutex     *sync.RWMutex
}

func newConsulWatcher(cfg *consul.Config, target string) (*consulWatcher, error) {
	c, err := consul.NewClient(cfg)
	if err != nil {
		logger.Errorf("fail to connect to consul: %v", err)
		return nil, err
	}
	consulClient := csd.NewClient(c)

	w := &consulWatcher{
		updateC: make(chan *updateMsg, 1),
		mutex:   &sync.RWMutex{},
	}
	w.instancer = instancer.NewInstancer(consulClient, sdLogger{}, target, nil, true,
		func(instances []string, tagMap map[string][]string, err error) {
			msg := &updateMsg{
				instances: instances,
				tagMap:    tagMap,
				err:       err,
			}
			logger.Infof("listener recv update msg %v", msg)
			w.updateC <- msg
		})

	return w, nil
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call should get the full set of the results. It should
// return an error if and only if Watcher cannot recover.
func (c *consulWatcher) Next() ([]*naming.Update, error) {
	logger.Info("watcher.Next() ...")
	select {
	case msg, ok := <-c.updateC:
		if ok {
			return c.makeUpdates(msg)
		}

		logger.Error("naming chan closed")
		return nil, fmt.Errorf("closed")
	}
}

func (c *consulWatcher) makeUpdates(msg *updateMsg) ([]*naming.Update, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	updates := make([]*naming.Update, 0)

	is := msg.instances
	tm := msg.tagMap
	err := msg.err
	if err != nil {
		logger.Errorf("Fail to watch updates: %v", err)
		return updates, nil
	}

	logger.Infof("current entries: %v", c.entries)
	if c.entries == nil {
		// first update, add all entries
		logger.Info("create new entries")
		c.entries = make(map[string]bool)
		for _, i := range is {
			u := &naming.Update{
				Op:       naming.Add,
				Addr:     i,
				Metadata: strings.Join(tm[i], ","),
			}
			updates = append(updates, u)
			c.entries[i] = true
		}
	} else {
		ne := make(map[string]bool)
		toadd := make([]string, 0)
		todel := make([]string, 0)

		for _, i := range is {
			if _, contains := c.entries[i]; contains {
				// existing entries
				ne[i] = true
				continue
			} else {
				// added entries
				u := &naming.Update{
					Op:       naming.Add,
					Addr:     i,
					Metadata: strings.Join(tm[i], ","),
				}
				updates = append(updates, u)
				toadd = append(toadd, i)
				ne[i] = true
			}
		}

		// deleted entries
		for i := range c.entries {
			if _, contains := ne[i]; contains {
				continue
			} else {
				u := &naming.Update{
					Op:   naming.Delete,
					Addr: i,
				}
				updates = append(updates, u)
				todel = append(todel, i)
			}
		}

		// update entries
		for _, i := range toadd {
			c.entries[i] = true
		}
		for _, i := range todel {
			delete(c.entries, i)
		}
	}

	for _, u := range updates {
		logger.WithFields(logrus.Fields{
			"event":     "consul_naming",
			"operation": u.Op,
			"address":   u.Addr,
			"tags":      u.Metadata,
		}).Info("naming update")
	}

	return updates, nil
}

func (c *consulWatcher) Close() {
	logger.Infof("consul_naming closed")
	c.instancer.SetListener(nil)
}

/*
type Update struct {
	// Op indicates the operation of the update.
	Op Operation
	// Addr is the updated address. It is empty string if there is no address update.
	Addr string
	// Metadata is the updated metadata. It is nil if there is no metadata update.
	// Metadata is not required for a custom naming implementation.
	Metadata interface{}
}
*/
