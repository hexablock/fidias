package main

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

func main() {
	cli := &CLI{}
	cli.Run()
}
