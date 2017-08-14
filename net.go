package fidias

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/hexablock/hexatype"
)

type rpcOutConn struct {
	host   string
	conn   *grpc.ClientConn
	client FidiasRPCClient
	used   time.Time
}

// NetTransport implements a network transport needed for fidias
type NetTransport struct {
	kvs KeyValueStore

	mu   sync.RWMutex
	pool map[string]*rpcOutConn

	maxConnIdle  time.Duration
	reapInterval time.Duration
	shutdown     int32
}

func NewNetTransport(kvs KeyValueStore, reapInterval, maxIdle time.Duration) *NetTransport {
	return &NetTransport{
		kvs:          kvs,
		pool:         make(map[string]*rpcOutConn),
		maxConnIdle:  maxIdle,
		reapInterval: reapInterval,
	}
}

// GetKey retrieves a key from a remote host
func (trans *NetTransport) GetKey(host string, key []byte) (*hexatype.KeyValuePair, error) {
	conn, err := trans.getConn(host)
	if err != nil {
		return nil, err
	}

	kvp, err := conn.client.GetKeyRPC(context.Background(), &hexatype.KeyValuePair{Key: key})
	if err != nil {
		err = hexatype.ParseGRPCError(err)
	}
	return kvp, err
}

// GetKeyRPC serves a GetKey request
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *hexatype.KeyValuePair) (*hexatype.KeyValuePair, error) {
	return trans.kvs.Get(in.Key)
}

// Shutdown signals the transport to be shutdown.  After shutdown no new connections can
// be made
func (trans *NetTransport) Shutdown() {
	atomic.StoreInt32(&trans.shutdown, 1)
}

func (trans *NetTransport) getConn(host string) (*rpcOutConn, error) {
	if atomic.LoadInt32(&trans.shutdown) == 1 {
		return nil, fmt.Errorf("transport is shutdown")
	}

	// Check if we have a conn cached
	trans.mu.RLock()
	if out, ok := trans.pool[host]; ok && out != nil {
		defer trans.mu.RUnlock()
		return out, nil
	}
	trans.mu.RUnlock()

	// Make a new connection
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	trans.mu.Lock()
	out := &rpcOutConn{
		host:   host,
		client: NewFidiasRPCClient(conn),
		conn:   conn,
		used:   time.Now(),
	}
	trans.pool[host] = out
	trans.mu.Unlock()

	return out, nil
}

func (trans *NetTransport) returnConn(o *rpcOutConn) {
	if atomic.LoadInt32(&trans.shutdown) == 1 {
		o.conn.Close()
		return
	}

	// Update the last used time
	o.used = time.Now()

	// Push back into the pool
	trans.mu.Lock()
	trans.pool[o.host] = o
	trans.mu.Unlock()
}

func (trans *NetTransport) reapOld() {
	for {
		if atomic.LoadInt32(&trans.shutdown) == 1 {
			return
		}
		time.Sleep(trans.reapInterval)
		trans.reapOnce()
	}
}

func (trans *NetTransport) reapOnce() {
	trans.mu.Lock()

	for host, conn := range trans.pool {
		if time.Since(conn.used) > trans.maxConnIdle {
			conn.conn.Close()
			delete(trans.pool, host)
		}
	}

	trans.mu.Unlock()
}
