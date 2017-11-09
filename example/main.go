package main

//
// var (
// 	dataDir = flag.String("data-dir", "./tmp", "Data directory")
// 	// gossip - TCP/UDP
// 	advAddr = flag.String("adv-addr", "127.0.0.1:43210", "Gossip advertise addr")
// 	// blox and dht address - TCP/UDP
// 	dhtAdvAddr = flag.String("dht-adv-addr", "127.0.0.1:12345", "DHT advertise addr")
// 	// GRPC address
// 	grpcAdvAddr = flag.String("grpc-adv-addr", "127.0.0.1:22345", "DHT advertise addr")
// 	// rest - HTTP
// 	httpAdvAddr = flag.String("http-addr", "127.0.0.1:9090", "HTTP advertise addr")
// 	// gossip addresses
// 	joinAddr = flag.String("join", "", "Existing servers to join via gossip")
// )

// func parseAddr(host string) (string, int) {
// 	host, port, _ := net.SplitHostPort(host)
// 	i, _ := strconv.ParseInt(port, 10, 32)
// 	return host, int(i)
// }
//
// func initConfig() *fidias.Config {
// 	if *dataDir == "" {
// 		log.Fatal("[ERROR] Data directory required!")
// 	}
//
// 	var err error
//
// 	c := fidias.DefaultConfig()
// 	c.DataDir, err = filepath.Abs(*dataDir)
// 	if err != nil {
// 		log.Fatal("[ERROR]", err)
// 	}
//
// 	conf := memberlist.DefaultLANConfig()
// 	conf.Name = *dhtAdvAddr
// 	host, port := parseAddr(*advAddr)
//
// 	// conf.GossipInterval = 50 * time.Millisecond
// 	// conf.ProbeInterval = 500 * time.Millisecond
// 	// conf.ProbeTimeout = 250 * time.Millisecond
// 	// conf.SuspicionMult = 1
//
// 	conf.LogOutput = ioutil.Discard
// 	conf.AdvertiseAddr = host
// 	conf.AdvertisePort = port
// 	conf.BindAddr = host
// 	conf.BindPort = port
//
// 	c.Memberlist = conf
//
// 	c.DHT = kelips.DefaultConfig(*dhtAdvAddr)
// 	c.DHT.Meta["hexalog"] = *grpcAdvAddr
//
// 	c.Hexalog = hexalog.DefaultConfig(*grpcAdvAddr)
// 	c.Hexalog.Votes = 2
//
// 	c.SetHashFunc(sha256.New)
// 	return c
// }

// func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//
// 		// TODO(tamird): point to merged gRPC code rather than a PR.
// 		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
// 		//if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
// 		if r.ProtoMajor == 2 {
// 			log.Printf("GRPC request='%+v'", r)
// 			grpcServer.ServeHTTP(w, r)
// 		} else {
// 			log.Printf("REGULAR request='%+v'", r)
// 			otherHandler.ServeHTTP(w, r)
// 		}
// 	})
// }

func main() {
	cli := &CLI{}
	cli.Run()

	// log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	// log.SetLevel("DEBUG")
	//
	// conf := initConfig()
	//
	// // Fidias
	// fid, err := fidias.Create(conf)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// if *joinAddr != "" {
	// 	if err = fid.Join([]string{*joinAddr}); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	//
	// restHandler := &httpServer{
	// 	dht: fid.DHT(),
	// 	kvs: fid.KVS(),
	// 	dev: fid.BlockDevice(),
	// }
	//
	// if err = http.ListenAndServe(*httpAdvAddr, restHandler); err != nil {
	// 	log.Fatal(err)
	// }

}
