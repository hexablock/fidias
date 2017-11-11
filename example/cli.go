package main

import (
	"crypto/sha256"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
	// Run agent vs a client
	isAgent = flag.Bool("agent", false, "Start agent")
	// gossip addresses
	joinAddr = flag.String("join", os.Getenv("FID_PEERS"), "Existing servers to join via gossip")
	// debug mode
	debug = flag.Bool("debug", false, "Turn debug mode on")
)

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

	if err = http.ListenAndServe(*httpAdvAddr, restHandler); err != nil {
		log.Fatal(err)
	}
}

func (cli *CLI) initAgentConfig(datadir string) *fidias.Config {
	c := fidias.DefaultConfig()
	c.DataDir = datadir

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

	c.Memberlist = conf

	c.DHT = kelips.DefaultConfig(*dataAdvAddr)
	c.DHT.Meta["hexalog"] = *grpcAdvAddr

	c.Hexalog = hexalog.DefaultConfig(*grpcAdvAddr)
	c.Hexalog.Votes = 2

	c.SetHashFunc(sha256.New)
	return c
}

//
// func (cli *CLI) initClient() *fidias.Client {
//
// 	if *joinAddr == "" {
// 		fmt.Println("FID_PEERS environment variable not set")
// 		os.Exit(1)
// 	}
//
// 	conf := fidias.DefaultConfig()
// 	conf.Peers = strings.Split(*joinAddr, ",")
// 	client, err := fidias.NewClient(conf)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	return client
// }

// func (cli *CLI) runClient(args []string) {
// 	client := cli.initClient()
// 	kvs := client.KVS()
//
// 	cmd := args[0]
//
// 	switch cmd {
//
// 	case "ls":
// 		cli.runClientLs(kvs, args[1])
//
// 	case "cp":
// 		cli.runClientCp(args[1])
//
// 	default:
// 		fmt.Println("Invalid command:", cmd)
// 		os.Exit(1)
// 	}
//
// 	//fmt.Println(args)
//
// }

// func (cli *CLI) runClientLs(kvs *fidias.KVS, dirname string) {
// 	opt := &fidias.ReadOptions{}
// 	kvps, _, err := kvs.List([]byte(dirname), opt)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(2)
// 	}
//
// 	fmt.Println(kvps)
// }
//
// func (cli *CLI) runClientCp(filename string) {
//
// }

func (cli *CLI) Run() {
	flag.Parse()

	if *debug {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
		log.SetLevel("DEBUG")
	} else {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.SetLevel("INFO")
	}

	// if *isAgent {
	// 	cli.runAgent()
	// } else {
	// 	cli.runClient(flag.Args())
	// }
	cli.runAgent()

}

func parseAddr(host string) (string, int) {
	host, port, _ := net.SplitHostPort(host)
	i, _ := strconv.ParseInt(port, 10, 32)
	return host, int(i)
}
