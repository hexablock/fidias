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
	"github.com/hexablock/fidias/gateways"
	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
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

	// Blox net
	bloxLn, err := net.Listen("tcp", *bloxAddr)
	if err != nil {
		log.Fatalf("[ERROR] Failed start listening on %s: %v", *bloxAddr, err)
	}

	// GRPC Server
	ln, err := net.Listen("tcp", *bindAddr)
	if err != nil {
		log.Fatalf("[ERROR] Failed start listening on %s: %v", *bindAddr, err)
	}

	gserver := grpc.NewServer()

	timeout := 3 * time.Second
	maxIdle := 3 * time.Minute
	reapInt := 30 * time.Second

	// Init hexaring
	peers := hexaring.NewInMemPeerStore()
	chordTrans := chord.NewGRPCTransport(timeout, maxIdle)
	ring := hexaring.New(conf.Ring, peers, chordTrans)
	ring.RegisterServer(gserver)

	// Init log store with fsm
	index, entries, stable, fsm, err := setupStores(conf, *dataDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load stored: %v", err)
	}
	logstore := hexalog.NewLogStore(entries, index, conf.Hexalog.Hasher)

	lognet := hexalog.NewNetTransport(reapInt, maxIdle)
	hexalog.RegisterHexalogRPCServer(gserver, lognet)

	hexlog, err := fidias.NewHexalog(conf, logstore, stable, fsm, lognet)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize hexalog: %v", err)
	}

	log.Println("[INFO] Hexalog initialized")

	// Key-value
	keyvs := fidias.NewKeyvs(conf.KeyValueNamespace, hexlog, fsm)
	log.Println("[INFO] Keyvs initialized")

	// Blox
	journal, bdev, err := setupBlockDevice(*dataDir, conf.Hasher())
	if err != nil {
		log.Fatalf("[ERROR] Failed to setup block device: %v", err)
	}
	bloxTrans := setupBlockDeviceTransport(bloxLn, bdev, conf.Hasher())

	// Fetcher
	fet := fidias.NewFetcher(index, entries, conf.Hexalog.Votes, conf.RelocateBufSize, conf.Hasher())
	fet.RegisterBlockTransport(bloxTrans)

	// Relocator
	rel := fidias.NewRelocator(int64(conf.Hexalog.Votes), conf.Hasher())
	rel.RegisterKeylogIndex(index)
	rel.RegisterBlockJournal(journal)

	blockReplicas := 2
	dev := fidias.NewRingDevice(blockReplicas, conf.Hasher(), bdev, bloxTrans)
	log.Printf("[INFO] Default RingDevice replicas=%d", blockReplicas)

	// Fidias
	fidTrans := fidias.NewNetTransport(fsm, index, journal, reapInt, maxIdle, conf.Hexalog.Votes, conf.Hasher())
	fidias.RegisterFidiasRPCServer(gserver, fidTrans)

	fids := fidias.New(conf, hexlog, fsm, rel, fet, keyvs, dev, fidTrans)

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
	log.Printf("[INFO] Starting API server bind-address=%s", *httpAddr)
	httpServer := gateways.NewHTTPServer("/v1", conf, ring, keyvs, logstore, dev, fids)
	http.Handle("/v1/", httpServer)

	// HTTP UI
	if conf.UIDir != "" {
		log.Printf("[INFO] Starting UI directory='%s'", conf.UIDir)
		fs := http.FileServer(http.Dir(conf.UIDir))
		http.Handle("/", http.StripPrefix("/", fs))
	} else {
		log.Printf("[INFO] UI disabled")
	}

	if err = http.ListenAndServe(*httpAddr, nil); err != nil {
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
