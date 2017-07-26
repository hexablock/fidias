package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/log"
)

var (
	version   string
	buildtime string
)

var (
	clusterAddr   = flag.String("cluster-addr", "127.0.0.1:54321", "Cluster bind address")
	httpAddr      = flag.String("http-addr", "127.0.0.1:9090", "HTTP bind address")
	joinAddr      = flag.String("join", "", "Comma delimted list of existing peers to join")
	retryJoinAddr = flag.String("retry-join", "", "Comma delimted list of existing peers to retry")
	debug         = flag.Bool("debug", false, "Turn no debug mode")
)

func printStartBanner(conf *fidias.Config) {
	fmt.Printf(`
  Cluster : %s
  Hasher  : %s
  HTTP    : %s

`, conf.Hostname(), conf.Hexalog.Hasher.Algorithm(), *httpAddr)
}

func configure(conf *fidias.Config) {
	if *debug {
		log.SetLevel("DEBUG")
		log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)

		// Lower the stabilization time in debug mode
		conf.Ring.StabilizeMin = 1 * time.Second
		conf.Ring.StabilizeMax = 3 * time.Second
	}

	conf.Ring.Meta["http"] = []byte(*httpAddr)

	printStartBanner(conf)
}

func initHexaring(conf *fidias.Config, peerStore hexaring.PeerStore, server *grpc.Server) (ring *hexaring.Ring, err error) {
	if *joinAddr != "" {
		addPeersToStore(peerStore, *joinAddr)
		ring, err = hexaring.Join(conf.Ring, peerStore, server)
	} else if *retryJoinAddr != "" {
		addPeersToStore(peerStore, *retryJoinAddr)
		ring, err = hexaring.RetryJoin(conf.Ring, peerStore, server)
	} else {
		ring, err = hexaring.Create(conf.Ring, server)
	}

	return ring, err
}

func init() {
	// Silence grpc
	grpclog.SetLogger(log.New(ioutil.Discard, "", 0))
}

func main() {
	flag.Parse()

	// Configuration
	conf := fidias.DefaultConfig(*clusterAddr)
	configure(conf)

	// Server
	ln, err := net.Listen("tcp", conf.Hostname())
	if err != nil {
		log.Fatal("[ERROR]", err)
	}
	gserver := grpc.NewServer()

	// Stores
	stableStore := &hexalog.InMemStableStore{}
	logStore := hexalog.NewInMemLogStore(conf.Hexalog.Hasher)

	peerStore := hexaring.NewInMemPeerStore()

	// Application FSM
	fsm := fidias.NewKeyValueFSM()

	// Fidias
	fids, err := fidias.New(conf, fsm, logStore, stableStore, gserver)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	// Start serving network requests as this is needed in order to init hexaring
	go gserver.Serve(ln)

	// Create or join chord ring
	log.Println("[INFO] Initializing ring ...")
	ring, err := initHexaring(conf, peerStore, gserver)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	// Register ring with Guac
	fids.Register(ring)

	// Start HTTP API
	log.Printf("[INFO] Starting HTTP server ...")
	httpServer := fidias.NewHTTPServer("/v1", fsm, logStore, fids)
	if err = http.ListenAndServe(*httpAddr, httpServer); err != nil {
		log.Fatal("[ERROR]", err)
	}

}

// addPeersToStore adds the given comma delimited addrs to the peer store
func addPeersToStore(peerStore hexaring.PeerStore, addrs string) {
	peers := strings.Split(addrs, ",")
	for _, peer := range peers {
		if p := strings.TrimSpace(peer); p != "" {
			peerStore.AddPeer(p)
		}
	}
}
