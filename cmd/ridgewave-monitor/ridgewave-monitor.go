package main

import (
	"fmt"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/reiver/go-telnet"
)

func main() {
	var args struct {
		Modem string
	}
	arg.MustParse(&args)

	conn, err := telnet.DialTo(args.Modem)
	if err != nil {
		log.Fatal(err)
	}
	_ = conn
	fmt.Println("done")
}
