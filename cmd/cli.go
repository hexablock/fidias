package main

import (
	"flag"
	"fmt"
	"hash"
	baselog "log"
	"os"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

var (
	// This is the address used by other cluster members to communicate with one-another
	advAddr = flag.String("adv-addr", os.Getenv("FIDS_ADV_ADDR"), "Cluster address to advertise [env FIDS_ADV_ADDR]")
	// This is what the cluster listen on i.e. can accept connections on this address space
	bindAddr = flag.String("bind-addr", "127.0.0.1:32100", "Cluster bind address")
	httpAddr = flag.String("http-addr", "127.0.0.1:7700", "HTTP bind address")
	// Address used by http server for redirects.  This is what is published in the meta for a vnode
	httpAdvAddr = flag.String("http-adv-addr", os.Getenv("FIDS_HTTP_ADV_ADDR"), "HTTP address to adversise [env FIDS_HTTP_ADV_ADDR]")

	bloxAddr    = flag.String("blox-addr", "127.0.0.1:42100", "Blox bind address")
	bloxAdvAddr = flag.String("blox-adv-addr", os.Getenv("FIDS_BLOX_ADV_ADDR"), "Blox bind address [env FIDS_BLOX_ADV_ADDR]")

	joinAddr      = flag.String("join", "", "Comma delimted list of existing peers to join")
	retryJoinAddr = flag.String("retry-join", "", "Comma delimted list of existing peers to retry joining")

	hashFunc = flag.String("hash", "SHA256", "Hash function to use [ SHA1 | SHA256 ]")
	dataDir  = flag.String("data-dir", os.Getenv("FIDS_DATA_DIR"), "Path to data directory for persistence [env FIDS_DATA_DIR]")

	showVersion = flag.Bool("version", false, "Show version")
	debug       = flag.Bool("debug", false, "Turn on debug mode")
)

func checkAddrs() {
	// Cluster advertise
	adv, err := buildAdvertiseAddr(*advAddr, *bindAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	*advAddr = adv

	// Http advertise
	hAdv, err := buildAdvertiseAddr(*httpAdvAddr, *httpAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	*httpAdvAddr = hAdv

	// Blox advertise
	bAdv, err := buildAdvertiseAddr(*bloxAdvAddr, *bloxAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	*bloxAdvAddr = bAdv
}

func configure() *fidias.Config {
	// Get config using cluster advertise address
	conf := fidias.DefaultConfig(*advAddr)
	// advertise address for http
	conf.Ring.Meta["http"] = []byte(*httpAdvAddr)
	conf.Ring.Meta["blox"] = []byte(*bloxAdvAddr)

	if *debug {
		// Setup the standard built-in log for underlying libraries
		baselog.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		baselog.SetPrefix(fmt.Sprintf("|%s| ", *advAddr))

		// Setup hexablock/log
		log.SetLevel(log.LogLevelDebug)
		log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		log.SetPrefix(fmt.Sprintf("|%s| ", *advAddr))

	} else {
		baselog.SetFlags(log.Lmicroseconds | log.LstdFlags)
		log.SetFlags(log.Lmicroseconds | log.LstdFlags)
		log.SetLevel(log.LogLevelInfo)
		log.SetPrefix(fmt.Sprintf("|%s| ", *advAddr))
	}

	// Set the hasher to sha256
	if *hashFunc == "SHA256" {
		conf.Hexalog.Hasher = &hexatype.SHA256Hasher{}
		conf.Ring.HashFunc = func() hash.Hash {
			return (&hexatype.SHA256Hasher{}).New()
		}
	}

	return conf
}

func printStartBanner(conf *fidias.Config) {
	fmt.Printf(`
  Version   : %s

  Advertise : %s
  Cluster   : %s
  Blox      : %s
  HTTP      : %s

  Hasher    : %s
  Vnodes    : %d

`, version, *advAddr, *bindAddr, *bloxAddr, *httpAddr, conf.Hexalog.Hasher.Algorithm(), conf.Ring.NumVnodes)
}
