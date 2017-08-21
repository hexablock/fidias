package main

import (
	"flag"
	"fmt"
	"hash"
	"io/ioutil"
	baselog "log"
	"net"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/hexablock/fidias"
	"github.com/hexablock/fidias/gateways"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

var (
	version   string
	buildtime string
)

var (
	advAddr       = flag.String("adv-addr", "", "Address to advertise to the network")
	clusterAddr   = flag.String("cluster-addr", "127.0.0.1:54321", "Cluster bind address")
	httpAddr      = flag.String("http-addr", "127.0.0.1:9090", "HTTP bind address")
	joinAddr      = flag.String("join", "", "Comma delimted list of existing peers to join")
	retryJoinAddr = flag.String("retry-join", "", "Comma delimted list of existing peers to retry")
	hashFunc      = flag.String("hash", "SHA1", "Hash function to use [ SHA1 | SHA256 ]")
	showVersion   = flag.Bool("version", false, "Show version")
	debug         = flag.Bool("debug", false, "Turn on debug mode")
)

func printStartBanner(conf *fidias.Config) {
	fmt.Printf(`
  Version   : %s
  Advertise : %s
  Cluster   : %s
  Hasher    : %s
  HTTP      : %s

`, version, *advAddr, conf.Hostname(), conf.Hexalog.Hasher.Algorithm(), *httpAddr)
}

func configure(conf *fidias.Config) {
	conf.Ring.Meta["http"] = []byte(*httpAddr)

	if *debug {
		// Setup the standard built-in log for underlying libraries
		baselog.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		baselog.SetPrefix(fmt.Sprintf("|%s| ", *clusterAddr))

		// Setup hexablock/log
		log.SetLevel(log.LogLevelDebug)
		log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		log.SetPrefix(fmt.Sprintf("|%s| ", *clusterAddr))

		// Lower the stabilization time in debug mode
		conf.Ring.StabilizeMin = 1 * time.Second
		conf.Ring.StabilizeMax = 3 * time.Second
	} else {
		baselog.SetFlags(log.Lmicroseconds | log.LstdFlags)
		log.SetFlags(log.Lmicroseconds | log.LstdFlags)
		log.SetLevel(log.LogLevelInfo)
	}

	// Set the hasher to sha256
	if *hashFunc == "SHA256" {
		conf.Hexalog.Hasher = &hexatype.SHA256Hasher{}
		conf.Ring.HashFunc = func() hash.Hash {
			return (&hexatype.SHA256Hasher{}).New()
		}
	}

	printStartBanner(conf)
}

func initHexaring(r *hexaring.Ring, peerStore hexaring.PeerStore) (err error) {
	switch {

	case *joinAddr != "":
		addPeersToStore(peerStore, *joinAddr)
		err = r.Join()

	case *retryJoinAddr != "":
		addPeersToStore(peerStore, *retryJoinAddr)
		err = r.RetryJoin()

	default:
		err = r.Create()

	}

	return
}

func init() {
	// Silence grpc
	grpclog.SetLogger(log.New(ioutil.Discard, "", 0))
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version, buildtime)
		return
	}

	if *advAddr == "" {
		// TODO: check it is a valid advertiseable address
		advAddr = clusterAddr
	}

	// Configuration
	conf := fidias.DefaultConfig(*advAddr)
	configure(conf)

	// Server
	ln, err := net.Listen("tcp", *clusterAddr)
	if err != nil {
		log.Fatalf("[ERROR] Failed start listening on %s: %v", *clusterAddr, err)
	}
	gserver := grpc.NewServer()

	// Stores
	stableStore := &store.InMemStableStore{}
	entries := store.NewInMemEntryStore()
	idxStore := store.NewInMemIndexStore()
	logStore := hexalog.NewLogStore(entries, idxStore, conf.Hexalog.Hasher)

	peers := hexaring.NewInMemPeerStore()
	// Init hexaring
	ring := hexaring.New(conf.Ring, peers, gserver)

	// Application FSM
	fsm := fidias.NewInMemKeyValueFSM()

	// Fidias
	fids, err := fidias.New(conf, fsm, idxStore, entries, logStore, stableStore, gserver)
	if err != nil {
		log.Fatal("[ERROR] Failed to initialize fidias:", err)
	}

	// Start serving network requests.  This needs to be started before trying to create or
	// join the ring as the ring initialization requires the transport
	go gserver.Serve(ln)

	// Create or join chord ring
	log.Printf("[INFO] Initializing ring bind-address=%s", *clusterAddr)
	if err = initHexaring(ring, peers); err != nil {
		log.Fatal("[ERROR]", err)
	}

	// Register ring
	fids.Register(ring)

	// Start HTTP API
	log.Printf("[INFO] Starting HTTP server bind-address=%s", *httpAddr)
	httpServer := gateways.NewHTTPServer("/v1", conf, ring, fsm, logStore, fids)
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
