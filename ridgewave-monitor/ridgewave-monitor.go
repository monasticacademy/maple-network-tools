package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
	"unicode"

	"github.com/alexflint/go-arg"
	"github.com/alexflint/go-restructure"
	"github.com/kr/pretty"
	"github.com/reiver/go-telnet"
)

// LTELine is for matching lines that look like this:
//    PCI(343) CID(A2F03) RSRP(-91) RSRQ(-8.8) RSSI(-65) SINR(30.0) RxLev(0)
type LTELine struct {
	_     struct{} `regexp:"PCI\\("`
	PCI   string   `regexp:".+"`
	_     struct{} `regexp:"\\) CID\\("`
	CID   string   `regexp:".+"`
	_     struct{} `regexp:"\\) RSRP\\("`
	RSRP  string   `regexp:".+"`
	_     struct{} `regexp:"\\) RSRQ\\("`
	RSRQ  string   `regexp:".+"`
	_     struct{} `regexp:"\\) RSSI\\("`
	RSSI  string   `regexp:".+"`
	_     struct{} `regexp:"\\) SINR\\("`
	SINR  string   `regexp:".+"`
	_     struct{} `regexp:"\\) RxLev\\("`
	RxLev string   `regexp:".+"`
	_     struct{} `regexp:"\\)"`
}

var pattern = restructure.MustCompile(LTELine{}, restructure.Options{})

// LTEInfo is the result of parsing integers from strings in LTELine
type LTEInfo struct {
	PCI   int
	CID   string
	RSRP  int
	RSRQ  float64
	RSSI  int
	SINR  float64
	RxLev int
}

func infoFromLine(in LTELine) (*LTEInfo, error) {
	var err error
	var out LTEInfo
	out.PCI, err = strconv.Atoi(in.PCI)
	if err != nil {
		return nil, fmt.Errorf("error parsing PCI: %w", err)
	}

	out.CID = in.CID

	out.RSRP, err = strconv.Atoi(in.RSRP)
	if err != nil {
		return nil, fmt.Errorf("error parsing RSRP: %w", err)
	}

	out.RSRQ, err = strconv.ParseFloat(in.RSRQ, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing RSRQ: %w", err)
	}

	out.RSSI, err = strconv.Atoi(in.RSSI)
	if err != nil {
		return nil, fmt.Errorf("error parsing RSSI: %w", err)
	}

	out.SINR, err = strconv.ParseFloat(in.SINR, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing SINR: %w", err)
	}

	out.RxLev, err = strconv.Atoi(in.RxLev)
	if err != nil {
		return nil, fmt.Errorf("error parsing RxLev: %w", err)
	}

	return &out, nil
}

func main() {
	var args struct {
		Modem string
	}
	args.Modem = "ridgewave.maple.cml.me:23"
	arg.MustParse(&args)

	conn, err := telnet.DialTo(args.Modem)
	if err != nil {
		log.Fatal(err)
	}
	_ = conn

	conn.Write([]byte("admin\r\n"))
	time.Sleep(time.Second)
	conn.Write([]byte("admin\r\n"))
	time.Sleep(time.Second)
	conn.Write([]byte("wan lte lteinfo\r\n"))
	time.Sleep(time.Second)

	fmt.Println("reading...")
	buf := make([]byte, 1)
	var line bytes.Buffer
	for {
		_, err := conn.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		c := buf[0]
		if c == '\r' || c == '\n' {
			var match LTELine
			if pattern.Find(&match, line.String()) {
				info, err := infoFromLine(match)
				if err != nil {
					log.Println(err)
					continue
				}
				pretty.Println(info)
			}
			line.Reset()
		} else if unicode.IsPrint(rune(c)) {
			line.WriteByte(c)
		}

		//os.Stdout.Write(buf[:n])
		//os.Stdout.Write(buf[:n])
		//fmt.Printf("read %d bytes\n", n)
	}

	fmt.Println("done")
}
