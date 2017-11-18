package main

import (
	"crypto/sha256"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/hashicorp/memberlist"
	"github.com/hexablock/fidias"
	"github.com/hexablock/fidias/gateway"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
)

func (cli *CLI) runAgent() error {
	err := initAdvertiseAddresses()
	if err != nil {
		return err
	}

	conf := cli.initAgentConfig()

	fid, err := fidias.Create(conf)
	if err != nil {
		return err
	}

	if *joinAddr != "" {
		if err = fid.Join([]string{*joinAddr}); err != nil {
			return err
		}
	} else if *retryJoinAddr != "" {
		if err = fid.RetryJoin([]string{*retryJoinAddr}); err != nil {
			return err
		}
	} else {
		log.Println("[INFO] Bootstrap node")
	}

	restHandler := &gateway.HTTPServer{
		DHT:    fid.DHT(),
		KVS:    fid.KVS(),
		Device: fid.BlockDevice(),
	}

	return http.ListenAndServe(*httpAddr, restHandler)
}

func (cli *CLI) initAgentConfig() *fidias.Config {
	if *dataDir == "" {
		log.Fatal("[ERROR] Data directory required!")
	}
	datadir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	c := fidias.DefaultConfig()
	c.Phi.DataDir = datadir

	c.Phi.Memberlist = initMemberlistConf()

	c.Phi.DHT = kelips.DefaultConfig(*dataAdvAddr)
	c.Phi.DHT.Meta["hexalog"] = *grpcAdvAddr

	c.Phi.Hexalog = hexalog.DefaultConfig(*grpcAdvAddr)
	c.Phi.Hexalog.Votes = 2

	c.Phi.SetHashFunc(sha256.New)
	return c
}

func initMemberlistConf() *memberlist.Config {
	conf := memberlist.DefaultLANConfig()
	conf.Name = *dataAdvAddr
	host, port := parseAddr(*gossipAdvAddr)

	conf.LogOutput = ioutil.Discard
	conf.AdvertiseAddr = host
	conf.AdvertisePort = port
	conf.BindAddr = host
	conf.BindPort = port

	return conf
}

func initAdvertiseAddresses() error {
	var err error

	if *dataAdvAddr, err = buildAdvertiseAddr(*dataAdvAddr, *dataBindAddr); err != nil {
		return err
	}

	if *grpcAdvAddr, err = buildAdvertiseAddr(*grpcAdvAddr, *grpcBindAddr); err != nil {
		return err
	}
	//
	// if *httpAdvAddr, err = buildAdvertiseAddr(*httpAdvAddr, *httpBindAddr); err != nil {
	// 	return err
	// }
	//
	*gossipAdvAddr, err = buildAdvertiseAddr(*gossipAdvAddr, *gossipBindAddr)

	return err
}
