package grpc

import (
	"fmt"

	"google.golang.org/grpc/naming"
)

type directResolver struct {
	naming.Resolver
	addr string
}

type directWatcher struct {
	addr    string
	updateC chan int
	updated bool
}

func newDirectResolver(addr string) naming.Resolver {
	return &directResolver{
		addr: addr,
	}
}

func (c *directResolver) Resolve(target string) (naming.Watcher, error) {
	w := &directWatcher{
		addr:    target,
		updateC: make(chan int, 1),
		updated: false,
	}

	go func() {
		w.updateC <- 0
	}()

	return w, nil
}

func (c *directWatcher) Next() (updates []*naming.Update, err error) {
	select {
	case _, ok := <-c.updateC:
		if ok {
			if c.updated {
				logger.Info("direct naming return 0 update")
				return
			}

			updates = make([]*naming.Update, 1)
			u := &naming.Update{
				Op:   naming.Add,
				Addr: c.addr,
			}
			updates[0] = u
			c.updated = true
			logger.Infof("direct naming add %s", c.addr)

			return
		}

		logger.Error("naming chan closed")
		return nil, fmt.Errorf("closed")
	}

}

func (c *directWatcher) Close() {
	close(c.updateC)
}
