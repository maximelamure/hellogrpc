package main

import (
	"io"
	"log"
	"math/rand"
	"strconv"
	"time"

	pb "github.com/maximelamure/hellogrpc/helloworld"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:50051"
	defaultName = "max"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewHelloClient(conn)

	ServerSideStreaming(c)
	ClientSideStreaming(c)
	ClientBiDirectionalStreaming(c)
}

func ServerSideStreaming(c pb.HelloClient) {
	s, err := c.GetServerSideStreaming(context.Background(), newFunckyUser(1))
	if err != nil {
		log.Fatalf("GetServerSideStreaming error %v", err)
	}
	for {
		p, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListFeatures(_) = _, %v", p, err)
		}
		log.Println(p)
	}
}

func ClientSideStreaming(c pb.HelloClient) {
	stream, err := c.GetClientSideStreaming(context.Background())
	if err != nil {
		log.Fatalf("%v.RecordRoute(_) = _, %v", c, err)
	}

	users := getUsers()
	for user := range users {
		if err := stream.Send(user); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, user, err)
		}
	}

	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Printf("Product: %v", reply)
}

func ClientBiDirectionalStreaming(c pb.HelloClient) {
	stream, err := c.GetBiDirectionalStreaming(context.Background())
	if err != nil {
		log.Fatalf("%v.RecordRoute(_) = _, %v", c, err)
	}

	q := make(chan bool)
	// Send users to the servers
	go func() {
		users := getUsers()
		for user := range users {
			if err := stream.Send(user); err != nil {
				log.Fatalf("%v.Send(%v) = %v", stream, user, err)
			}
		}

		if err := stream.CloseSend(); err != nil {
			log.Println("error to close: ", err)
		}
		q <- true
	}()

	// Receive products from the server
	for {
		p, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("%v.ListFeatures(_) = _, %v", p, err)
		}

		log.Println(p, "received")
	}

	<-q
}

func newFunckyUser(index int) *pb.User {
	return &pb.User{Name: "User_" + strconv.Itoa(index)}
}

func getUsers() chan *pb.User {
	ch := make(chan *pb.User)
	go func() {
		nbP := 20
		log.Printf("Push %d user", nbP)
		for i := 0; i < nbP; i++ {
			p := newFunckyUser(i)
			log.Println(p, "pushed")
			ch <- p
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
		}

		close(ch)
	}()

	return ch
}
