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

var shard = flag.Int("shard", 0, "The shard number")

func (s *server) Ping(ctx context.Context, in *proto.PingRequest) (*proto.PingResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	return &proto.PingResponse{Message: "Pong"}, nil
}

func (s *server) ScheduleWorkflow(ctx context.Context, in *proto.ScheduleWorkflowRequest) (*proto.ScheduleWorkflowResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	return &proto.ScheduleWorkflowResponse{Shard: int32(*shard), Message: fmt.Sprintf("Scheduled %d", *shard)}, nil
}

func (s *server) ReportTaskResult(ctx context.Context, in *proto.ReportTaskResultRequest) (*proto.ReportTaskResultResponse, error) {
	log.Printf("Request: %s\n", in.Message)
	if in.Shard != int32(*shard) {
		return &proto.ReportTaskResultResponse{Message: "Shard mismatch"}, nil
	}
	return &proto.ReportTaskResultResponse{Message: "Reported"}, nil
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
