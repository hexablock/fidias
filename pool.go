package fidias

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

type rpcOutConn struct {
	host   string
	conn   *grpc.ClientConn
	client FidiasRPCClient
	used   time.Time
}

type outPool struct {
	mu   sync.RWMutex
	pool map[string]*rpcOutConn

	maxConnIdle  time.Duration
	reapInterval time.Duration
	stopped      int32
}

func newOutPool(maxIdle, reapInterval time.Duration) *outPool {
	return &outPool{
		maxConnIdle:  maxIdle,
		reapInterval: reapInterval,
		pool:         make(map[string]*rpcOutConn),
	}
}

func (pool *outPool) returnConn(o *rpcOutConn) {
	// Close and discard connection if we've shutdown
	if atomic.LoadInt32(&pool.stopped) == 1 {
		o.conn.Close()
		return
	}

	// Update the last used time
	o.used = time.Now()

	// Push back into the pool
	pool.mu.Lock()
	pool.pool[o.host] = o
	pool.mu.Unlock()
}

func (pool *outPool) getConn(host string) (*rpcOutConn, error) {
	if atomic.LoadInt32(&pool.stopped) == 1 {
		return nil, fmt.Errorf("transport is shutdown")
	}

	// Check if we have a conn cached
	pool.mu.RLock()
	if out, ok := pool.pool[host]; ok && out != nil {
		defer pool.mu.RUnlock()
		return out, nil
	}
	pool.mu.RUnlock()

	// Make a new connection
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	pool.mu.Lock()
	out := &rpcOutConn{
		host:   host,
		client: NewFidiasRPCClient(conn),
		conn:   conn,
		used:   time.Now(),
	}
	pool.pool[host] = out
	pool.mu.Unlock()

	return out, nil
}

func (pool *outPool) reapOld() {
	for {
		if atomic.LoadInt32(&pool.stopped) == 1 {
			return
		}
		time.Sleep(pool.reapInterval)
		pool.reapOnce()
	}
}

func (pool *outPool) reapOnce() {
	pool.mu.Lock()

	for host, conn := range pool.pool {
		if time.Since(conn.used) > pool.maxConnIdle {
			conn.conn.Close()
			delete(pool.pool, host)
		}
	}

	pool.mu.Unlock()
}

func (pool *outPool) shutdown() {
	atomic.StoreInt32(&pool.stopped, 1)
}
