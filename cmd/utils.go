package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexablock/blox"
	"github.com/hexablock/blox/device"
	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexatype"
)

func setupBlockDeviceTransport(ln net.Listener, localDev *device.BlockDevice, hasher hexatype.Hasher) *blox.LocalTransport {

	opts := blox.DefaultNetClientOptions(hasher)
	remote := blox.NewNetTransport(ln, opts)

	// TODO: change to use adv-address for blox
	trans := blox.NewLocalTransport(ln.Addr().String(), remote)
	trans.Register(localDev)

	log.Printf("[INFO] BlockDevice transport bind-address=%s", ln.Addr().String())
	return trans
}

func setupBlockDevice(basedir string, hasher hexatype.Hasher) (device.Journal, *device.BlockDevice, error) {
	dir := filepath.Join(basedir, "blox", "blocks")
	os.MkdirAll(dir, 0755)

	rdev, err := device.NewFileRawDevice(dir, hasher)
	if err != nil {
		return nil, nil, err
	}

	journal := device.NewInmemJournal()
	dev := device.NewBlockDevice(journal, rdev)

	log.Println("[INFO] BlockDevice journal=in-memory raw-device=file")
	return journal, dev, nil
}

func setupStores(conf *fidias.Config, baseDir string) (index hexalog.IndexStore, entries hexalog.EntryStore,
	stable hexalog.StableStore, fsm *fidias.FSM, err error) {

	log.Printf("[INFO] Using ephemeral storage: in-memory")
	entries = hexalog.NewInMemEntryStore()
	index = hexalog.NewInMemIndexStore()
	stable = &hexalog.InMemStableStore{}
	fsm = fidias.NewFSM(conf.Namespaces.KeyValue, conf.Namespaces.FileSystem)
	return
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
