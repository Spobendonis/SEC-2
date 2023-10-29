package main

import (
	"bufio"
	"context"
	"flag"
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
	isHospital = flag.Bool("isHospital", false, "Are you the hospital?")
	hospitalId = 7000
	numPeers   = len(os.Args) - 2
	p          = &peer{
		id:           0,
		secret:       0,
		hospital:     nil,
		clients:      make([]Hospital.PeerClient, len(os.Args)-2),
		shares:       make([]int, numPeers),
		summedShares: 0,
		ctx:          nil,
	}
	recievedShares = 0
)

// TODO: unspaghettify this code -> do something smart and readable for the hospital
func main() {
	flag.Parse()
	//If the isHospital flag is true, this value will not be overwritten, making the port = hospitalId
	var port = hospitalId
	var peers = make([]int, numPeers)
	fmt.Printf("%d\n", numPeers)
	//If the isHospital flag is false, set the port to (hospitalId + the first arg)
	if !*isHospital {
		var x, _ = strconv.Atoi(os.Args[1])
		port = hospitalId + x
	}
	//Turn the relative ports into absolute ports (adding the offset to the hospitalPort) and store it in peers[]
	for i := 0; i < numPeers; i++ {
		var x, _ = strconv.Atoi(os.Args[i+2])
		peers[i] = hospitalId + x
	}
	log.Printf("my port is %d", port)
	log.Printf("my peers are %d", peers)

	//Grpc set up
	ctx, cancel := context.WithCancel(context.Background())
	p.id = int32(port)
	p.ctx = ctx
	defer cancel()

	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}

	//Creat a certificate using self-signed cer and key
	//openssl req -nodes -x509 -sha256 -newkey rsa:4096 -keyout priv.key -out server.crt -days 356 -subj "/C=DK/ST=Copenhagen/L=Copenhagen/O=Me/OU=mpc/CN=localhost" -addext "subjectAltName = DNS:localhost,IP:0.0.0.0"
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

	//This uses the array of ports, peers[] to create and array of Clients
	for i, e := range peers {
		var conn = createTlsConn(e)
		defer conn.Close()
		p.clients[i] = Hospital.NewPeerClient(conn)
		fmt.Printf("%v", p.clients)
	}
	//This creates the hospital client, saves it in the hospital field
	var conn = createTlsConn(hospitalId)
	defer conn.Close()
	p.hospital = Hospital.NewPeerClient(conn)
	scanner := bufio.NewScanner(os.Stdin)
	//If you are a peer wait for cmd input for what your secret is
	if !*isHospital {
		fmt.Println("Enter a number between 0 and 1 000 000 to share it secretly with the other peers.\nNumber: ")
		for scanner.Scan() {
			secret, _ := strconv.ParseInt(scanner.Text(), 10, 32)
			share(int(secret))
		}
		//Else you are the hospital, simply wait until you have all the secrets
	} else {
		fmt.Println("Waiting for data from peers...\nwrite 'quit' to end me")
		for scanner.Scan() {
			if scanner.Text() == "quit" {
				return
			}
		}
	}
}

func share(secret int) {
	log.Println("Sharing secret!")
	//uses splitSecret function to split the secret, and stores the value in an array
	var shares = splitSecret(secret)
	//For each share from the secret, send the share to a different peer
	for i, client := range p.clients {
		fmt.Printf("%d: %d\n", i, shares[i])
		var _, _ = client.SendShare(p.ctx, &Hospital.Share{Value: int32(shares[i])})
	}
}

// Creates and returns a TLS connection given a specific port
func createTlsConn(port int) *grpc.ClientConn {
	clientCert, err := credentials.NewClientTLSFromFile("certificate/server.crt", "")
	if err != nil {
		log.Fatalln("failed to create cert", err)
	}

	fmt.Printf("Trying to dial: %v\n", port)
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithTransportCredentials(clientCert), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	return conn
}

// This is not the most clever way to do this, but it basically choses alternating
// random numbers (once positive, then negative). Then the last number is a correction
// value that makes sure the numbers sum to the secret.

// TA HELP: I think theres a clever way to do this using Group Theory? but i dont know how
func splitSecret(secret int) []int {
	//Randomize the source
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

// this is ran in the server part of the peer, when a different client has called the SendShare on their respective client
func (c *peer) SendShare(ctx context.Context, in *Hospital.Share) (*Hospital.Empty, error) {
	if recievedShares < numPeers {
		log.Printf("Recieved: %d\n", in.Value)
		//Adds the recieved share to the array of all recieved shares
		c.shares[recievedShares] += int(in.Value)
		//increments the number of shares recieved, used to know when you have all shares
		recievedShares++
		log.Printf("Shares: %o", recievedShares)
		//if you have all shares, sums all the shares together
		if recievedShares >= numPeers {
			for _, e := range c.shares {
				c.summedShares += e
			}
			log.Printf("Sum: %d", c.summedShares)
			//Sends the sum of shares to the hospital
			c.hospital.SendSummedShares(c.ctx, &Hospital.Share{Value: int32(c.summedShares)})
		}
	}
	return &Hospital.Empty{}, nil
}

// Should only be run on the hospital
func (c *peer) SendSummedShares(ctx context.Context, in *Hospital.Share) (*Hospital.Empty, error) {
	//checks if it is the hospital, and if it is, adds the recieved share (which is a sum of shares sent from a client)
	//and increments the number of shares recieved
	if int(c.id) == hospitalId {
		c.summedShares += int(in.Value)
		recievedShares++
		//if it has all the shares, it prints it for you :)
		if recievedShares >= numPeers {
			log.Printf("%d\n", c.summedShares)
		}
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
