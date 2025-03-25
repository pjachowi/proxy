package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"foobar/proxy/proto"

	"google.golang.org/grpc"
)

type server struct {
	proto.UnimplementedPingServer
}

func (s *server) Ping(ctx context.Context, in *proto.PingRequest) (*proto.PingResponse, error) {
	// log.Printf("Request: %s\n", in.Message)
	// time.Sleep(time.Millisecond * 100)
	return &proto.PingResponse{Message: "Pong"}, nil
}

var port = flag.Int("port", 50050, "The server port")

func main() {
	log.Println("Hello, world!")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterPingServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
