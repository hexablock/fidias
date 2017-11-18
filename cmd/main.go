package main

import "os"

func usage() {
	data := []byte(`
Usage: fid [-version] [-h|--help] [-debug] [ options ]

Fidias is a distributed and decentralized datastore with no node being special

Agent:

  -agent [ options ]                Run the fidias agent

    -data-dir <directory>           Data directory
    -gossip-addr <address:port>     Gossip advertise address
    -data-addr <address:port>       Data and DHT advertise address
    -rpc-addr <address:port>        GRPC advertise address
    -join <peer1,peer2>             List of peers to join
    -retry-join <peer1,peers>       List of peers to retry joins

Client (experimental):

  set <key> <value>    Set a key-value pair
  get <key>            Get a key
  rm  <key>            Remove a key
  ls  <prefix>         List a prefix

`)

	os.Stderr.Write(data)
}

//
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

var (
	version   string
	buildtime string
)

func main() {
	cli := &CLI{}
	cli.Run()
}
