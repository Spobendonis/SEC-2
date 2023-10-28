package main

import (
	// "context"
	// "errors"
	// "flag"
	// "fmt"
	// "log"
	// "math/rand"
	// "net"
	// "os"
	// "strconv"
	// "time"

	// Hospital "github.com/Spobendonis/Sec-2/grpc"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	Hospital "github.com/Spobendonis/Sec-2/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	hospitalId = 5000
	p          = &peer{
		id:      5000,
		clients: make([]Hospital.PeerClient, len(os.Args)),
		ctx:     nil,
	}
)

func main() {
	var x, _ = strconv.Atoi(os.Args[1])
	var port = hospitalId + x
	var peers = make([]int, len(os.Args)-2)
	for i := 2; i < len(os.Args); i++ {
		x, _ = strconv.Atoi(os.Args[i])
		peers[i-2] = hospitalId + x
	}
	log.Printf("my port is %d", port)
	log.Printf("my peers are %d", peers)

	ctx, cancel := context.WithCancel(context.Background())
	p.id = int32(port)
	p.ctx = ctx
	defer cancel()

	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}

	serverCert, err := credentials.NewServerTLSFromFile("certificate/server.crt", "certificate/priv.key")
	if err != nil {
		log.Fatalln("failed to create cert", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(serverCert))
	Hospital.RegisterPeerServer(grpcServer, p)

	// start the server
	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()

	for i, e := range peers {
		clientCert, err := credentials.NewClientTLSFromFile("certificate/server.crt", "")
		if err != nil {
			log.Fatalln("failed to create cert", err)
		}

		fmt.Printf("Trying to dial: %v\n", e)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", e), grpc.WithTransportCredentials(clientCert), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()
		c := Hospital.NewPeerClient(conn)
		p.clients[i] = c
		fmt.Printf("%v", p.clients)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if port != hospitalId {
		fmt.Print("Enter a number between 0 and 1 000 000 to share it secretly with the other peers.\nNumber: ")
		for scanner.Scan() {
			secret, _ := strconv.ParseInt(scanner.Text(), 10, 32)
			//p.Share(int(secret))
			log.Printf("%d", int(secret))
		}
	} else {
		fmt.Print("Waiting for data from peers...\nwrite 'quit' to end me\n")
		for scanner.Scan() {
			if scanner.Text() == "quit" {
				return
			}
		}
	}
}

/*func Share(secret int) {
	SendShare
}

func (c *peerClient) SendShare(ctx context.Context, in *Share, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/ping.Peer/sendShare", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}*/

type peer struct {
	Hospital.UnimplementedPeerServer
	id      int32
	clients []Hospital.PeerClient
	ctx     context.Context
}
