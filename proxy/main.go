package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"foobar/proxy/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type server struct {
	proto.UnimplementedPingServer
}

var port = flag.Int("port", 50051, "The server port")
var master = flag.String("master", "localhost:50050", "The master server")
var retryPolicy = `{
	"methodConfig": [{
	  "retryPolicy": {
		  "MaxAttempts": 14,
		  "InitialBackoff": ".01s",
		  "MaxBackoff": "1s",
		  "BackoffMultiplier": 1.4,
		  "RetryableStatusCodes": [ "UNAVAILABLE" ]
	  }
	}]}`

var clients map[string]proto.PingClient

func (s *server) Ping(_ context.Context, in *proto.PingRequest) (*proto.PingResponse, error) {
	// log.Printf("Request: %s\n", in.Message)
	c := clients[*master]

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return c.Ping(ctx, &proto.PingRequest{Message: "Ping"})
}

func main() {
	flag.Parse()
	log.Println("Hello, world!")
	clients = make(map[string]proto.PingClient)
	conn, err := grpc.NewClient(
		*master,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithConnectParams(
			grpc.ConnectParams{
				Backoff: backoff.Config{
					BaseDelay:  1 * time.Second,
					Multiplier: 1.4,
					Jitter:     0.2,
					MaxDelay:   1 * time.Second,
				},
				MinConnectTimeout: 1 * time.Second,
			}))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	clients[*master] = proto.NewPingClient(conn)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterPingServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
