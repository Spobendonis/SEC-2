# Sec-1 Hand-in 2

## Certificate

The certificate and private key were generated from commandline using:

    'openssl req -nodes -x509 -sha256 -newkey rsa:4096 -keyout priv.key -out server.crt -days 356 -subj "/C=DK/ST=Copenhagen/L=Copenhagen/O=Me/OU=mpc/CN=localhost" -addext "subjectAltName = DNS:localhost,IP:0.0.0.0"'

You can try to regenerate these if you have trouble with the certificate

## How to run
In order to run the simulation, you have to open at least 4 terminal windows (This is also the minimal expectation for this project). For every terminal you have to run the main.go file, with n - 1 args (where n is the number of terminals opened). These arguments correspond to ports, relative to a base port (the default base port is 7000). The first argument is the relative port where the process is hosted. Each following argument is the relative port of a connecting process (aka a peer).
One terminal has to represent the hospital. In order to do this, you must set the 'isHospital' flag to 'true'. Each argument is then the relative port of the peers connected to the hospital.
### Example:

Terminal 1(Hospital):
```c
//This opens the hospital at port 7000(default), which tries to establish connection to ports 7001, 7002, 7003
go run main.go --isHospital=true 1 2 3
```

Terminal 2(Peer1):
```c
//This opens a peer at port 7001, which tries to establish connection to ports 7000(Each peer automatically tries to connect to the default port to establish connection with the hospital), 7002, 7003
go run main.go 1 2 3
```

Terminal 3(Peer2):
```c
//This opens a peer at port 7002, which tries to establish connection to ports 7000(Each peer automatically tries to connect to the default port to establish connection with the hospital), 7001, 7003
go run main.go 2 1 3
```

Terminal 4(Peer3):
```c
//This opens a peer at port 7003, which tries to establish connection to ports 7000(Each peer automatically tries to connect to the default port to establish connection with the hospital), 7001, 7002
go run main.go 3 1 2
```

Once you have run the processes, it will prompt you with your secret. You can write this in the command line, and it will send it to all other ports. this is done in text input instead of piplining such that it is easier to verify that it works as expected. 

## Additional points

This code should support operating with more than 3 peers. In theory it should be able to go up to as many as you want. this functionality is howevere not tested since it is out of scope for this assignement.