package main

import (
	"flag"
	"fmt"
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
	"github.com/hexablock/hexaring"
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
		log.SetPrefix(fmt.Sprintf("|%s| ", *clusterAddr))

		// Lower the stabilization time in debug mode
		conf.Ring.StabilizeMin = 1 * time.Second
		conf.Ring.StabilizeMax = 3 * time.Second
	} else {
		log.SetFlags(log.Lmicroseconds | log.LstdFlags)
		log.SetLevel(log.LogLevelInfo)
	}

	printStartBanner(conf)
}

func initHexaring(conf *fidias.Config, peerStore hexaring.PeerStore, server *grpc.Server) (ring *hexaring.Ring, err error) {
	switch {

	case *joinAddr != "":
		addPeersToStore(peerStore, *joinAddr)
		ring, err = hexaring.Join(conf.Ring, peerStore, server)

	case *retryJoinAddr != "":
		addPeersToStore(peerStore, *retryJoinAddr)
		ring, err = hexaring.RetryJoin(conf.Ring, peerStore, server)

	default:
		ring, err = hexaring.Create(conf.Ring, peerStore, server)

	}

	return ring, err
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
	stableStore := &hexalog.InMemStableStore{}
	logStore := hexalog.NewInMemLogStore(conf.Hexalog.Hasher)

	peerStore := hexaring.NewInMemPeerStore()

	// Application FSM
	fsm := fidias.NewInMemKeyValueFSM()

	// Fidias
	fids, err := fidias.New(conf, fsm, logStore, stableStore, gserver)
	if err != nil {
		log.Fatal("[ERROR] Failed to initialize fidias:", err)
	}

	// Start serving network requests as this is needed in order to init hexaring
	go gserver.Serve(ln)

	// Create or join chord ring
	log.Printf("[INFO] Initializing ring bind-address=%s", *clusterAddr)
	ring, err := initHexaring(conf, peerStore, gserver)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	// Register ring with Guac
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
