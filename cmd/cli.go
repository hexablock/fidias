package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hexablock/fidias"
	"github.com/hexablock/log"
)

var (
	dataDir = flag.String("data-dir", os.Getenv("FID_DATADIR"), "Data directory")
	// gossip - TCP/UDP
	gossipBindAddr = flag.String("gossip-bind-addr", "127.0.0.1:32100", "Gossip bind addr")
	gossipAdvAddr  = flag.String("gossip-adv-addr", os.Getenv("FID_GOSSIP_ADV_ADDR"), "Gossip advertise addr")
	// blox and dht address - TCP/UDP
	dataBindAddr = flag.String("data-bind-addr", "127.0.0.1:42100", "DHT and block bind addr")
	dataAdvAddr  = flag.String("data-adv-addr", os.Getenv("FID_DATA_ADV_ADDR"), "DHT and block advertise addr")
	// GRPC address
	grpcBindAddr = flag.String("rpc-bind-addr", "127.0.0.1:8800", "RPC bind addr")
	grpcAdvAddr  = flag.String("rpc-adv-addr", os.Getenv("FID_RPC_ADV_ADDR"), "RPC advertise addr")
	// HTTP rest gateway
	httpAddr = flag.String("http-addr", "127.0.0.1:9090", "HTTP gateway addrress")
	//httpAdvAddr  = flag.String("http-adv-addr", os.Getenv("FID_HTTP_ADV_ADDR"), "HTTP gateway advertise addr")

	// Gossip addresses for agent and gRPC addresses for clients
	joinAddr      = flag.String("join", os.Getenv("FID_PEERS"), "Existing peers to join via gossip")
	retryJoinAddr = flag.String("retry-join", os.Getenv("FID_RETRY_PEERS"), "Existing peers to join via gossip")

	isAgent   = flag.Bool("agent", false, "Run the agent")
	debug     = flag.Bool("debug", false, "Turn debug mode on")
	isVersion = flag.Bool("version", false, "Show version")
)

// CLI is the command line interface
type CLI struct {
}

func (cli *CLI) runClient(args []string) error {
	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	client, err := setupClient()
	if err != nil {
		return err
	}

	var (
		kvclient = client.KV()
		cmd      = args[0]
		key      = []byte(args[1])
		data     interface{}
	)

	switch cmd {
	case "get":
		data, _, err = kvclient.Get(key, &fidias.ReadOptions{})

	case "set":
		if len(args) != 3 {
			err = fmt.Errorf("not enough args")
			break
		}

		kvp := fidias.NewKVPair(key, []byte(args[2]))
		wo := fidias.DefaultWriteOptions()
		data, _, err = kvclient.Set(kvp, wo)

	case "rm":
		wo := fidias.DefaultWriteOptions()
		_, err = kvclient.Remove(key, wo)

	case "ls":
		if len(args) != 2 {
			err = fmt.Errorf("prefix not specified")
			break
		}
		data, _, err = kvclient.List([]byte(args[1]), &fidias.ReadOptions{})

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

func (cli *CLI) isVersion() {
	if *isVersion {
		fmt.Println(version)
		os.Exit(0)
	}
}

// Run parses and run the command line args
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

	cli.isVersion()

	if *isAgent {
		if err := cli.runAgent(); err != nil {
			log.Fatal("[ERROR]", err)
		}
		return
	}

	if err := cli.runClient(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(2)
	}

}

func setupClient() (*fidias.Client, error) {
	conf := fidias.DefaultConfig()
	// Grpc address
	if *joinAddr != "" {
		conf.Phi.Hexalog.AdvertiseHost = *joinAddr
	} else {
		// default
		conf.Phi.Hexalog.AdvertiseHost = *grpcAdvAddr
	}

	return fidias.NewClient(conf)
}
