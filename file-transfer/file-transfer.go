package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
)

type receiveArgs struct {
	Port int
}

type sendArgs struct {
	Path string `arg:"positional,required"`
	Dest string `arg:"positional,required"`
}

type args struct {
	Receive *receiveArgs `arg:"subcommand"`
	Send    *sendArgs    `arg:"subcommand"`
}

type uploadRequest struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

type uploadResponse struct {
	Success bool
	Error   string
}

func sendMain(ctx context.Context, args *args) error {
	st, err := os.Stat(args.Send.Path)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadFile(args.Send.Path)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", args.Send.Dest)
	if err != nil {
		return err
	}
	defer conn.Close()

	var b bytes.Buffer
	err = gob.NewEncoder(&b).Encode(uploadRequest{
		Path:    filepath.Base(args.Send.Path),
		Content: buf,
		Mode:    st.Mode(),
	})
	if err != nil {
		return fmt.Errorf("error gob-encoding upload request: %w", err)
	}

	n, err := b.WriteTo(conn)
	if err != nil {
		return fmt.Errorf("error writing payload to tcp connection: %w", err)
	}
	fmt.Printf("wrote %d bytes (of %d) using %v\n", n, b.Len(), conn.LocalAddr())

	err = conn.Close()
	if err != nil {
		return err
	}

	return nil
}

func receiveMain(ctx context.Context, args *args) error {
	// Listen for incoming connections
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", args.Receive.Port))
	if err != nil {
		return fmt.Errorf("error listening for tcp connections: %w", err)
	}
	defer l.Close()

	fmt.Printf("listening on %d...\n", args.Receive.Port)
	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("error accepting: ", err)
			continue
		}

		fmt.Println("accepted a connection from", conn.RemoteAddr())

		// Handle connections in a new goroutine.
		go handleRequest(ctx, conn)
	}
}

func handleRequest(ctx context.Context, conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Printf("reading from %v...\n", conn.RemoteAddr())

	var req uploadRequest
	err := gob.NewDecoder(conn).Decode(&req)
	if err != nil {
		fmt.Println("error decoding request:", err)
		return
	}

	fmt.Printf("received %s (%d bytes)\n", req.Path, len(req.Content))

	err = ioutil.WriteFile(req.Path, req.Content, req.Mode)
	if err != nil {
		fmt.Printf("error writing %d bytes to %s: %v\n", len(req.Content), req.Path, err)
		return
	}

	fmt.Printf("wrote %s\n", req.Path)

	// TODO: send a response
}

func main() {
	ctx := context.Background()

	var args args
	arg.MustParse(&args)

	var err error
	switch {
	case args.Receive != nil:
		err = receiveMain(ctx, &args)
	case args.Send != nil:
		err = sendMain(ctx, &args)
	default:
		fmt.Println("you must specify either send or receive")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
