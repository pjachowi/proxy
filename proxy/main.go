package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"foobar/proxy/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type server struct {
	proto.UnimplementedPingServer
	masters map[string]*grpc.ClientConn
}

var port = flag.Int("port", 50051, "The server port")
var masters = flag.String("master", "0:localhost:50050", "The master server")
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

var clients map[string]*grpc.ClientConn

func broadcast(s *server, ctx context.Context, method string, args, reply any) {
	for shard, c := range s.masters {
		c.Invoke(ctx, method, args, reply)
		log.Printf("Broadcasted to %s: %v\n", shard, reply)
	}
}

func (s *server) Ping(c context.Context, in *proto.PingRequest) (*proto.PingResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	broadcast(s, c, "/proto.Ping/Ping", in, &proto.PingResponse{})
	return &proto.PingResponse{Message: "Pong"}, nil
}

func (s *server) ScheduleWorkflow(ctx context.Context, in *proto.ScheduleWorkflowRequest) (*proto.ScheduleWorkflowResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	var err error
	keys := make([]string, 0, len(s.masters))
	for k := range s.masters {
		keys = append(keys, k)
	}
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	for _, shard := range keys {
		client := proto.NewPingClient(s.masters[shard])
		resp, err := client.ScheduleWorkflow(ctx, in)
		if err != nil {
			log.Printf("Error scheduling on shard: %s %v\n", shard, err)
			continue
		}
		log.Printf("Scheduled on shard: %s resp: %v\n", shard, resp)
		return resp, nil
	}
	return nil, err
}

func (s *server) ReportTaskResult(_ context.Context, in *proto.ReportTaskResultRequest) (*proto.ReportTaskResultResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	c := proto.NewPingClient(clients["0"])
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return c.ReportTaskResult(ctx, in)
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Hello, world!")
	clients = make(map[string]*grpc.ClientConn)
	for _, masterConfig := range strings.Split(*masters, ",") {
		m := strings.SplitN(masterConfig, ":", 2)
		if len(m) != 2 {
			log.Fatalf("invalid master config: %s", masterConfig)
		}
		conn, err := grpc.NewClient(
			m[1],
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
			// TODO: to be decided if we should continue if one of the masters is down
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		clients[m[0]] = conn
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterPingServer(s, &server{masters: clients})
	log.Printf("server listening at %v", lis.Addr())
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
