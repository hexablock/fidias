package fidias

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"
)

// KeyValueStore implements a key value store interface
type KeyValueStore interface {
	Get(key []byte) (*KeyValuePair, error)
}

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
	pool map[string][]*rpcOutConn

	maxConnIdle  time.Duration
	reapInterval time.Duration
	shutdown     int32
}

// GetKey retrieves a key from a remote host
func (trans *NetTransport) GetKey(host string, key []byte) (*KeyValuePair, error) {
	conn, err := trans.getConn(host)
	if err != nil {
		return nil, err
	}

	return conn.client.GetKeyRPC(context.Background(), &KeyValuePair{Key: key})
}

// GetKeyRPC serves a GetKey request
func (trans *NetTransport) GetKeyRPC(ctx context.Context, in *KeyValuePair) (*KeyValuePair, error) {
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
	var out *rpcOutConn
	trans.mu.Lock()
	list, ok := trans.pool[host]
	if ok && len(list) > 0 {
		out = list[len(list)-1]
		list = list[:len(list)-1]
		trans.pool[host] = list
	}
	trans.mu.Unlock()
	// Make a new connection
	if out == nil {
		conn, err := grpc.Dial(host, grpc.WithInsecure())
		if err == nil {
			return &rpcOutConn{
				host:   host,
				client: NewFidiasRPCClient(conn),
				conn:   conn,
				used:   time.Now(),
			}, nil
		}
		return nil, err
	}

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
	list, _ := trans.pool[o.host]
	trans.pool[o.host] = append(list, o)
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

	for host, conns := range trans.pool {
		max := len(conns)
		for i := 0; i < max; i++ {
			if time.Since(conns[i].used) > trans.maxConnIdle {
				conns[i].conn.Close()
				conns[i], conns[max-1] = conns[max-1], nil
				max--
				i--
			}
		}
		// Trim any idle conns
		trans.pool[host] = conns[:max]
	}

	trans.mu.Unlock()
}
