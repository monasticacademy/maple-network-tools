package main

import (
	"context"
	"log"
	"net"

	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
)

func main() {
	ctx := context.Background()

	var args struct {
		Hostport string `arg:"positional,required"`
		Project  string `arg:"required" help:"GCP project for logging"`
		LogName  string `arg:"required" help:"tag for GCP logs"`
	}
	arg.MustParse(&args)

	// Creates a client.
	logs, err := logging.NewClient(ctx, args.Project)
	if err != nil {
		log.Fatalf("error creating logging client: %v", err)
	}
	defer logs.Close()

	// Sets the name of the log to write to.
	logger := logs.Logger(args.LogName).StandardLogger(logging.Info)

	// listen for UDP packets
	addr, err := net.ResolveUDPAddr("udp", args.Hostport)
	if err != nil {
		log.Fatalf("error parsing hostport: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("error listening for UDP packets: %v", err)
	}

	logger.Printf("listening on %s\n", args.Hostport)
	buf := make([]byte, 1<<20)
	for {
		n, _, err := conn.ReadFrom(buf)

		// the docs say that valid data can be returned even when there is also an error
		if n > 0 {
			logger.Writer().Write(buf[:n])
		}

		if err != nil {
			logger.Printf("error reading UDP packet: %v (%T)\n", err, err)
		}
	}

	logger.Println("exiting")
}
