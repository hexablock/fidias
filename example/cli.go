package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/memberlist"
	"github.com/hexablock/fidias"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/log"
)

var (
	dataDir = flag.String("data-dir", "", "Data directory")
	// gossip - TCP/UDP
	gossipAdvAddr = flag.String("gossip-addr", "127.0.0.1:43210", "Gossip advertise addr")
	// blox and dht address - TCP/UDP
	dataAdvAddr = flag.String("data-addr", "127.0.0.1:12345", "DHT and block advertise addr")
	// GRPC address
	grpcAdvAddr = flag.String("rpc-addr", "127.0.0.1:22345", "RPC advertise addr")
	// rest - HTTP
	httpAdvAddr = flag.String("http-addr", "127.0.0.1:9090", "HTTP advertise addr")
	// gossip addresses
	joinAddr = flag.String("join", os.Getenv("FID_PEERS"), "Existing servers to join via gossip")
	// agent mode
	isAgent = flag.Bool("agent", false, "Run the agent")
	// debug mode
	debug = flag.Bool("debug", false, "Turn debug mode on")
)

func usage() {
	data := []byte(`
Usage: fid [ options ]

Fidias is a distributed and decentralized datastore with no node being special

Agent:

  -agent [ options ]                Run the fidias agent

    -data-dir <directory>           Data directory
    -gossip-addr <address:port>     Gossip advertise address
    -data-addr <address:port>       Data and DHT advertise address
    -rpc-addr <address:port>        GRPC advertise address
    -join <peer1,peer2>             List of peers to join

Client:

  set <key> <value>    Set a key-value pair
  get <key>            Get a key
  rm  <key>            Remove a key
  ls  <prefix>         List a prefix

`)

	os.Stderr.Write(data)
}

type CLI struct{}

func (cli *CLI) runAgent() {
	if *dataDir == "" {
		log.Fatal("[ERROR] Data directory required!")
	}

	datadir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	conf := cli.initAgentConfig(datadir)

	fid, err := fidias.Create(conf)
	if err != nil {
		log.Fatal(err)
	}

	if *joinAddr != "" {
		if err = fid.Join([]string{*joinAddr}); err != nil {
			log.Fatal(err)
		}
	}

	restHandler := &httpServer{
		dht: fid.DHT(),
		kvs: fid.KVS(),
		dev: fid.BlockDevice(),
	}

	// opts := []grpc.ServerOption{
	// 	grpc.Creds(credentials.NewClientTLSFromCert(demoCertPool, *httpAdvAddr)),
	// }
	// grpcServer := grpc.NewServer(opts...)
	//
	// srv := &http.Server{
	// 	Addr:    *httpAdvAddr,
	// 	Handler: grpcHandlerFunc(grpcServer, mux),
	// 	TLSConfig: &tls.Config{
	// 		Certificates: []tls.Certificate{*demoKeyPair},
	// 		NextProtos:   []string{"h2"},
	// 	},
	// }

	if err = http.ListenAndServe(*httpAdvAddr, restHandler); err != nil {
		log.Fatal(err)
	}
}

func (cli *CLI) initAgentConfig(datadir string) *fidias.Config {

	c := fidias.DefaultConfig()
	c.Phi.DataDir = datadir

	conf := memberlist.DefaultLANConfig()
	conf.Name = *dataAdvAddr
	host, port := parseAddr(*gossipAdvAddr)

	// conf.GossipInterval = 50 * time.Millisecond
	// conf.ProbeInterval = 500 * time.Millisecond
	// conf.ProbeTimeout = 250 * time.Millisecond
	// conf.SuspicionMult = 1

	conf.LogOutput = ioutil.Discard
	conf.AdvertiseAddr = host
	conf.AdvertisePort = port
	conf.BindAddr = host
	conf.BindPort = port

	c.Phi.Memberlist = conf

	c.Phi.DHT = kelips.DefaultConfig(*dataAdvAddr)
	c.Phi.DHT.Meta["hexalog"] = *grpcAdvAddr

	c.Phi.Hexalog = hexalog.DefaultConfig(*grpcAdvAddr)
	c.Phi.Hexalog.Votes = 2

	c.Phi.SetHashFunc(sha256.New)
	return c
}

func (cli *CLI) runClient() error {
	conf := fidias.DefaultConfig()
	conf.Peers = strings.Split(*joinAddr, ",")
	if len(conf.Peers) < 1 || conf.Peers[0] == "" {
		return fmt.Errorf("FID_PEERS env. variable not set")
	}
	conf.Phi.Hexalog.AdvertiseHost = *grpcAdvAddr

	client, err := fidias.NewClient(conf)
	if err != nil {
		return err
	}

	args := flag.Args()
	if len(args) < 2 {
		return fmt.Errorf("not enough args")
	}

	kv := client.KV()

	wo := fidias.DefaultWriteOptions()
	key := []byte(args[1])

	var data interface{}

	switch args[0] {
	case "get":
		data, _, err = kv.Get(key, &fidias.ReadOptions{})

	case "set":
		if len(args) != 3 {
			err = fmt.Errorf("not enough args")
			break
		}
		kvp := fidias.NewKVPair(key, []byte(args[2]))
		data, _, err = kv.Set(kvp, wo)

	case "rm":
		_, err = kv.Remove(key, wo)

	case "ls":
		if len(args) != 2 {
			err = fmt.Errorf("prefix not specified")
			break
		}
		data, _, err = kv.List([]byte(args[1]), &fidias.ReadOptions{})

	default:
		err = fmt.Errorf("command not found: %s", args[0])
	}

	if err != nil {
		return err
	}

	if data != nil {
		b, _ := json.MarshalIndent(data, "", "  ")
		os.Stdout.Write(b)
		os.Stdout.Write([]byte("\n"))
	}
	return nil
}

func (cli *CLI) Run() {
	flag.Usage = usage
	flag.Parse()

	if *debug {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetLevel("DEBUG")
	} else {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.SetLevel("INFO")
	}

	if *isAgent {
		cli.runAgent()
	} else {
		if err := cli.runClient(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(2)
		}
	}

}

func parseAddr(host string) (string, int) {
	host, port, _ := net.SplitHostPort(host)
	i, _ := strconv.ParseInt(port, 10, 32)
	return host, int(i)
}
