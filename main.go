package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	Hospital "github.com/Spobendonis/Sec-2/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TODO, make p.secret and hospitalId Flags
var (
	hospitalId = 5000
	numPeers   = len(os.Args) - 2
	p          = &peer{
		id:           5000,
		secret:       0,
		hospital:     nil,
		clients:      make([]Hospital.PeerClient, len(os.Args)-1),
		shares:       make([]int, numPeers),
		summedShares: 0,
		ctx:          nil,
	}
	recievedShares = 0
)

// TODO: unspaghettify this code -> do something smart and readable for the hospital
func main() {
	var x, _ = strconv.Atoi(os.Args[1])
	var port = hospitalId + x
	var peers = make([]int, numPeers+1)
	peers[0] = hospitalId
	for i := 1; i < numPeers+1; i++ {
		x, _ = strconv.Atoi(os.Args[i+1])
		peers[i] = hospitalId + x
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
	p.hospital = p.clients[0]
	p.clients = p.clients[1:]
	scanner := bufio.NewScanner(os.Stdin)
	if port != hospitalId {
		fmt.Println("Enter a number between 0 and 1 000 000 to share it secretly with the other peers.\nNumber: ")
		for scanner.Scan() {
			secret, _ := strconv.ParseInt(scanner.Text(), 10, 32)
			Share(int(secret))
		}
	} else {
		fmt.Println("Waiting for data from peers...\nwrite 'quit' to end me")
		for scanner.Scan() {
			if scanner.Text() == "quit" {
				return
			}
		}
	}
}

func Share(secret int) {
	log.Println("Sharing secret!")
	var shares = splitSecret(secret)
	for i, client := range p.clients {
		fmt.Printf("%d: %d\n", i, shares[i])
		var _, _ = client.SendShare(p.ctx, &Hospital.Share{Value: int32(shares[i])})
	}
}

// This is not the most clever way to do this, but it basically choses alternating
// random numbers (once positive, then negative). Then the last number is a correction
// value that makes sure the numbers sum to the secret.

// TA HELP: I think theres a clever way to do this using Group Theory? but i dont know how
func splitSecret(secret int) []int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	var res = make([]int, numPeers)
	var temp = 0
	var correction = secret
	for i := range res {
		if i < numPeers-1 {
			if i%2 == 0 {
				temp = r1.Intn(10000)
				correction -= temp
				res[i] = temp
			} else {
				temp = r1.Intn(10000) * -1
				correction -= temp
				res[i] = temp
			}
		} else {
			res[i] = correction
		}
	}
	return res
}

func (c *peer) SendShare(ctx context.Context, in *Hospital.Share) (*Hospital.Empty, error) {
	if recievedShares < numPeers {
		log.Printf("Recieved: %d\n", in.Value)
		c.shares[recievedShares] += int(in.Value)
		recievedShares++
		log.Printf("Shares: %o", recievedShares)
		if recievedShares >= numPeers {
			for _, e := range c.shares {
				c.summedShares += e
			}
			log.Printf("Sum: %d", c.summedShares)
			c.hospital.SendSummedShares(c.ctx, &Hospital.Share{Value: int32(c.summedShares)})
		}
	}
	return &Hospital.Empty{}, nil
}

func (c *peer) SendSummedShares(ctx context.Context, in *Hospital.Share) (*Hospital.Empty, error) {
	if int(c.id) == hospitalId {
		c.summedShares += int(in.Value)
		recievedShares++
	}
	if recievedShares >= numPeers {
		log.Printf("%d\n", c.summedShares)
	}
	return &Hospital.Empty{}, nil
}

type peer struct {
	Hospital.UnimplementedPeerServer
	id           int32
	secret       int
	hospital     Hospital.PeerClient
	clients      []Hospital.PeerClient
	shares       []int
	summedShares int
	ctx          context.Context
}
