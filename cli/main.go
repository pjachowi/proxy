package main

import (
	"context"
	"flag"
	"foobar/proxy/proto"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var proxy = flag.String("proxy", "localhost:50051", "The proxy address")

var numConcurrentRequests = flag.Int("concurrent", 1, "The number of concurrent requests")
var durationSec = flag.Int("duration", 60, "The duration of the test in seconds")

func call(c proto.PingClient) (*proto.PingResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return c.Ping(ctx, &proto.PingRequest{Message: "Ping"})
}

func main() {
	flag.Parse()
	conn, err := grpc.NewClient(*proxy, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := proto.NewPingClient(conn)

	ctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(*durationSec))
	start := time.Now()
	var numSuccess int
	var numFailures int
	guard := make(chan struct{}, *numConcurrentRequests)
loop:
	for {
		// log.Printf("Iteration %d\n", i)
		select {
		case <-ctx.Done():
			break loop
		default:
			guard <- struct{}{}
			go func() {
				_, err := call(c)
				if err != nil {
					// log.Println("Error:", err)
					numFailures++
				} else {
					// log.Println("Success")
					numSuccess++
				}
				<-guard
				// log.Printf("Iteration %d\n", success)
			}()
		}
	}
	elapsed := time.Since(start)
	log.Printf("success: %d failures: %d Average time: %s\n", numSuccess, numFailures, elapsed/time.Duration(numSuccess))

}
