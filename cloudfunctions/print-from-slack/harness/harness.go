// Run the cloud function locally

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
	p "github.com/alexflint/maple-network-tools/cloudfunctions/print-from-slack"
)

func main() {
	var args struct {
		Address string
	}
	args.Address = ":19870"
	arg.MustParse(&args)

	http.HandleFunc("/print-google-doc", p.PrintGoogleDoc)

	fmt.Println("listening on", args.Address)
	err := http.ListenAndServe(args.Address, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
