package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexablock/fidias"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexalog/store"
)

func setupStores(baseDir string) (index store.IndexStore, entries store.EntryStore,
	stable hexalog.StableStore, fsm fidias.KeyValueFSM, err error) {

	if baseDir == "" {
		log.Printf("[INFO] Using ephemeral storage: in-memory")
		index = store.NewInMemIndexStore()
		entries = store.NewInMemEntryStore()
		stable = &store.InMemStableStore{}
		fsm = fidias.NewInMemKeyValueFSM()
		return
	}

	log.Printf("[INFO] Using persistent storage: badger")
	idir := filepath.Join(baseDir, "index")
	edir := filepath.Join(baseDir, "entry")
	sdir := filepath.Join(baseDir, "stable")
	fdir := filepath.Join(baseDir, "fsm")
	os.MkdirAll(idir, 0755)
	os.MkdirAll(edir, 0755)
	os.MkdirAll(sdir, 0755)
	os.MkdirAll(fdir, 0755)

	idx := store.NewBadgerIndexStore(idir)
	if err = idx.Open(); err != nil {
		return
	}
	index = idx

	ents := store.NewBadgerEntryStore(edir)
	if err = ents.Open(); err != nil {
		idx.Close()
		return
	}
	entries = ents

	fsm = fidias.NewBadgerKeyValueFSM(fdir)
	stable = store.NewBadgerStableStore(sdir)

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
