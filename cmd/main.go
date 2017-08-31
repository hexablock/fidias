package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/hexablock/fidias"
	"github.com/hexablock/fidias/gateways"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/log"
)

var (
	version   string
	buildtime string
)

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

	checkAddrs()

	// Configuration
	conf := configure()
	printStartBanner(conf)

	// Server
	ln, err := net.Listen("tcp", *bindAddr)
	if err != nil {
		log.Fatalf("[ERROR] Failed start listening on %s: %v", *bindAddr, err)
	}
	gserver := grpc.NewServer()

	// Stores
	os.MkdirAll(*dataDir, 0755)
	index, entries, stable, err := setupStores(*dataDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load stored: %v", err)
	}
	logStore := hexalog.NewLogStore(entries, index, conf.Hexalog.Hasher)

	// Init hexaring
	peers := hexaring.NewInMemPeerStore()
	ring := hexaring.New(conf.Ring, peers, gserver)

	// Application FSM
	fsm := fidias.NewInMemKeyValueFSM()

	// Fidias
	fids, err := fidias.New(conf, fsm, index, entries, logStore, stable, gserver)
	if err != nil {
		log.Fatal("[ERROR] Failed to initialize fidias:", err)
	}

	// Start serving network requests.  This needs to be started before trying to create or
	// join the ring as the ring initialization requires the transport
	go gserver.Serve(ln)

	// Create or join chord ring
	log.Printf("[INFO] Initializing ring bind-address=%s", *bindAddr)
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

func setupStores(baseDir string) (index store.IndexStore, entries store.EntryStore, stable hexalog.StableStore, err error) {

	stable = &store.InMemStableStore{}

	if baseDir == "" {
		log.Printf("[INFO] Using ephemeral storage")
		index = store.NewInMemIndexStore()
		entries = store.NewInMemEntryStore()
		return
	}

	log.Printf("[INFO] Using persistent storage")
	idir := filepath.Join(baseDir, "index")
	edir := filepath.Join(baseDir, "entry")
	os.MkdirAll(idir, 0755)
	os.MkdirAll(edir, 0755)

	idx := store.NewBadgerIndexStore(idir)
	if err = idx.Open(); err != nil {
		return nil, nil, nil, err
	}
	index = idx

	ents := store.NewBadgerEntryStore(edir)
	if err = ents.Open(); err != nil {
		idx.Close()
		return nil, nil, nil, err
	}
	entries = ents

	return
}
