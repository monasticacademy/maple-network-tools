package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/mdlayher/wifi"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

//go:embed service-account.json
var googleCredentials []byte

func listStations() {
	wificlient, err := wifi.New()
	if err != nil {
		fmt.Printf("error creating wifi client: %v\n", err)
		os.Exit(1)
	}

	ifs, err := wificlient.Interfaces()
	if err != nil {
		fmt.Printf("error listing interfaces: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("found %d interfaces\n", len(ifs))

	for _, iface := range ifs {
		fmt.Println(iface.Name)
		infos, err := wificlient.StationInfo(iface)
		if err != nil {
			fmt.Println("error getting station info:g", err)
			continue
		}

		fmt.Printf("  got %d info structs\n", len(infos))

		for _, info := range infos {
			fmt.Printf("  %d\n", info.Signal)
		}

	}
}

// logEntry is the data that send to cloud logging once per N seconds
type logEntry struct {
	From   string
	To     string
	Bytes  int
	Frames int
}

// link represents a from and to address
type link struct {
	From string
	To   string
}

// statistics represents the statistics we keep for each from/to address
type statistics struct {
	Frames int // number of wifi frames observed
	Bytes  int // number of bytes in all observed wifi frames
}

func main() {
	ctx := context.Background()

	var args struct {
		Interface string `arg:"positional,required"`
		LogName   string
		Interval  time.Duration
	}
	args.LogName = "wifi-packets"
	args.Interval = 10 * time.Minute
	arg.MustParse(&args)

	// unpack google credentials
	creds, err := google.CredentialsFromJSON(ctx, googleCredentials)
	if err != nil {
		log.Fatal("error parsing credentials: ", err)
	}

	log.Println("project:", creds.ProjectID)
	log.Println("log interval:", args.Interval)

	// create the logger
	logClient, err := logging.NewClient(ctx, creds.ProjectID,
		option.WithCredentialsJSON(googleCredentials))
	if err != nil {
		log.Fatal("error creating logging client: ", err)
	}
	defer logClient.Close()

	lg := logClient.Logger(args.LogName)

	// open packet capture handle
	handle, err := pcapgo.NewEthernetHandle(args.Interface)
	if err != nil {
		log.Fatal("error creating ethernet handle: ", err)
	}

	lastUpload := time.Now()
	statsByLink := make(map[link]*statistics)

	pkgsrc := gopacket.NewPacketSource(handle, layers.LayerTypeDot11)
	for packet := range pkgsrc.Packets() {
		lay := packet.Layer(layers.LayerTypeDot11)
		if lay == nil {
			continue
		}

		wifiFrame, ok := lay.(*layers.Dot11)
		if !ok {
			continue
		}

		//fmt.Println(packet)
		//continue

		from := wifiFrame.Address1
		to := wifiFrame.Address2
		key := link{From: from.String(), To: to.String()}
		stats := statsByLink[key]
		if stats == nil {
			stats = new(statistics)
			statsByLink[key] = stats
		}

		stats.Frames += 1
		stats.Bytes += len(wifiFrame.Payload)

		now := time.Now()
		if now.Sub(lastUpload) > args.Interval {
			for k, v := range statsByLink {
				lg.Log(logging.Entry{
					Payload: logEntry{
						From:   k.From,
						To:     k.To,
						Frames: v.Frames,
						Bytes:  v.Bytes,
					},
				})
			}

			log.Printf("uploaded %d log entries\n", len(statsByLink))

			// TODO: delete the entries one-by-one to reduce memory churn
			statsByLink = make(map[link]*statistics)
			lastUpload = now
		}

	}

	_ = ctx
}
