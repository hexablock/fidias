package main

import (
	"flag"
	"fmt"
	"hash"
	baselog "log"
	"net"
	"os"
	"strings"
	"time"

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
	// Address used by http server for redirects
	httpAdvAddr = flag.String("http-adv-addr", os.Getenv("FIDS_HTTP_ADV_ADDR"), "HTTP address to adversise [env FIDS_ADV_ADDR]")

	joinAddr      = flag.String("join", "", "Comma delimted list of existing peers to join")
	retryJoinAddr = flag.String("retry-join", "", "Comma delimted list of existing peers to retry")
	hashFunc      = flag.String("hash", "SHA1", "Hash function to use [ SHA1 | SHA256 ]")

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
}

// given a advertise and bind address return the advertise addr or an error
func buildAdvertiseAddr(a, b string) (adv string, err error) {
	var addr string
	if a != "" {
		addr = a
	} else {
		// Used bind if adv is not supplied
		addr = b
	}

	parts := strings.Split(addr, ":")
	l := len(parts)
	if l > 1 {
		l--
		// Parse addr to make sure it is a usable ip address
		host := strings.Join(parts[:l], ":")
		var ipaddr *net.IPAddr
		ipaddr, err = net.ResolveIPAddr("ip", host)
		if err == nil {
			ip := ipaddr.String()
			port := parts[l]
			if port != "" && ip != "0.0.0.0" && ip != "::" && ip != "0:0:0:0:0:0:0:0" {
				adv = ip + ":" + port
				return
			}

		} else {
			return
		}
	}

	err = fmt.Errorf("Invalid advertise address: %s", addr)
	return
}

func configure() *fidias.Config {
	// Get config using cluster advertise address
	conf := fidias.DefaultConfig(*advAddr)
	// advertise address for http
	conf.Ring.Meta["http"] = []byte(*httpAdvAddr)

	if *debug {
		// Setup the standard built-in log for underlying libraries
		baselog.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		baselog.SetPrefix(fmt.Sprintf("|%s| ", *advAddr))

		// Setup hexablock/log
		log.SetLevel(log.LogLevelDebug)
		log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LstdFlags)
		log.SetPrefix(fmt.Sprintf("|%s| ", *advAddr))

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

	return conf
}

func printStartBanner(conf *fidias.Config) {
	fmt.Printf(`
  Version   : %s
  Advertise : %s
  Cluster   : %s
  Hasher    : %s
  HTTP      : %s

`, version, *advAddr, *bindAddr, conf.Hexalog.Hasher.Algorithm(), *httpAddr)
}
