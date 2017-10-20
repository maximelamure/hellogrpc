package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	pb "github.com/maximelamure/hellogrpc/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.HelloServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello" + in.Name}, nil
}

func (s *server) SayHelloAgain(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello Again" + in.Name}, nil
}

// Get with Server-Side Streaming
func (s *server) GetServerSideStreaming(u *pb.User, stream pb.Hello_GetServerSideStreamingServer) error {
	ch := getProducts()
	for p := range ch {
		if err := stream.Send(p); err != nil {
			log.Println("Error: ", err)
			return err
		}
	}
	return nil
}

func newFunckyProduct(index int) *pb.Product {
	return &pb.Product{Name: "Product_" + strconv.Itoa(index)}
}

func getProducts() chan *pb.Product {
	ch := make(chan *pb.Product)
	go func() {
		nbP := 20
		log.Printf("Push %d products", nbP)
		for i := 0; i < nbP; i++ {
			p := newFunckyProduct(i)
			log.Println(p, "pushed")
			ch <- p
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
		}

		close(ch)
	}()

	return ch
}

// Get with Client-Side Streaming
func (s *server) GetClientSideStreaming(stream pb.Hello_GetClientSideStreamingServer) error {
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(newFunckyProduct(1))
		}

		if err != nil {
			return err
		}

		fmt.Println("User received: ", user)
	}
}

// Get with bidirectional Streaming
func (s *server) GetBiDirectionalStreaming(stream pb.Hello_GetBiDirectionalStreamingServer) error {
	q := make(chan bool)

	// Stream from the client-> server
	go func() {
		for {
			user, err := stream.Recv()
			if err == io.EOF {
				q <- true
				break
			}
			if err != nil {
				log.Println("Error: ", err)
			}

			log.Println(user, " received")
		}
		q <- true
	}()

	// Stream from the server-> client
	ch := getProducts()
	for p := range ch {
		if err := stream.Send(p); err != nil {
			log.Println("Error: ", err)
		}
	}

	<-q
	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterHelloServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
