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
	ring := hexaring.New(conf.Ring, peers, timeout, maxIdle)
	ring.RegisterServer(gserver)

	// Init log store with fsm
	index, entries, stable, fsm, err := setupStores(*dataDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to load stored: %v", err)
	}
	logstore := hexalog.NewLogStore(entries, index, conf.Hexalog.Hasher)

	lognet := hexalog.NewNetTransport(reapInt, maxIdle)
	hexalog.RegisterHexalogRPCServer(gserver, lognet)

	hexlog, err := fidias.NewHexalog(conf.Hexalog, logstore, stable, fsm, lognet)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize hexalog: %v", err)
	}

	log.Println("[INFO] Hexalog initialized")

	// Key-value
	keyvs := fidias.NewKeyvs(hexlog, fsm)
	log.Println("[INFO] Keyvs initialized")

	// Blox
	journal, bdev, err := setupBlockDevice(*dataDir, conf.Hasher())
	if err != nil {
		log.Fatalf("[ERROR] Failed to setup block device: %v", err)
	}

	// Fetcher
	fet := fidias.NewFetcher(index, entries, conf.Hexalog.Votes, conf.RelocateBufSize)
	fet.RegisterBlockDevice(bdev)
	// Relocator
	rel := fidias.NewRelocator(int64(conf.Hexalog.Votes), conf.Hasher())
	rel.RegisterKeylogIndex(index)
	rel.RegisterBlockJournal(journal)

	bloxTrans := setupBlockDeviceTransport(bloxLn, bdev, conf.Hasher())

	blockReplicas := 2
	dev := fidias.NewRingDevice(blockReplicas, conf.Hasher(), bloxTrans)
	log.Println("[INFO] RingDevice replicas=%d", blockReplicas)

	// Fidias
	fidTrans := fidias.NewNetTransport(fsm, index, journal, reapInt, maxIdle, conf.Hexalog.Votes, conf.Hasher())
	fidias.RegisterFidiasRPCServer(gserver, fidTrans)

	fids := fidias.New(conf, hexlog, rel, fet, keyvs, dev, fidTrans)

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
	httpServer := gateways.NewHTTPServer("/v1", conf, ring, keyvs, logstore, dev, fids)
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
