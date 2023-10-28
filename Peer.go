package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)
func main() {
	log.SetFlags(log.Lshortfile)
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + hospitalId //clients are 5001 5002 5003

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:             ownPort,
		clients:        make(map[int32]exchange.ExchangeDataClient),
		receivedChunks: []int{},
		ctx:            ctx,
	}

	//set up server
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", ownPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverCert, err := credentials.NewServerTLSFromFile("certificate/server.crt", "certificate/priv.key")
	if err != nil {
		log.Fatalln("failed to create cert", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(serverCert))
	exchange.RegisterExchangeDataServer(grpcServer, p)

	// start the server
	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()

	//Dial the other peers
	for i := 0; i <= 3; i++ {
		port := hospitalId + int32(i)

		if port == ownPort {
			continue
		}

		//Set up client connections
		clientCert, err := credentials.NewClientTLSFromFile("certificate/server.crt", "")
		if err != nil {
			log.Fatalln("failed to create cert", err)
		}

		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithTransportCredentials(clientCert), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()
		c := exchange.NewExchangeDataClient(conn)
		p.clients[port] = c
		fmt.Printf("%v", p.clients)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if ownPort != hospitalId {
		fmt.Print("Enter a number between 0 and 1 000 000 to share it secretly with the other peers.\nNumber: ")
		for scanner.Scan() {
			secret, _ := strconv.ParseInt(scanner.Text(), 10, 32)
			p.ShareDataChunks(int(secret))
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