package main

import (
	"context"
	_ "embed"
	"log"
	"time"

	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

//go:embed service-account.json
var googleCredentials []byte

// logEntry is the data that send to cloud logging once per N seconds
type logEntry struct {
	From    string
	To      string
	Bytes   int64
	Packets int64
}

// link represents a from and to address
type link struct {
	From string
	To   string
}

// statistics represents the statistics we keep for each from/to address
type statistics struct {
	Packets int64 // number of packets observed
	Bytes   int64 // number of bytes in all observed wifi frames
}

func main() {
	ctx := context.Background()

	var args struct {
		Interface string `arg:"positional,required"`
		LogName   string
		Interval  time.Duration
	}
	args.LogName = "orbi-packets"
	args.Interval = 10 * time.Second
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
	statsBySource := make(map[string]*statistics)
	// statsByLink := make(map[link]*statistics)

	var packets, bytes int64
	pkgsrc := gopacket.NewPacketSource(handle, layers.LayerTypeDot11)
	for packet := range pkgsrc.Packets() {
		// fmt.Println("packet:")
		// for _, lay := range packet.Layers() {
		// 	fmt.Println(lay.LayerType())
		// }

		lay := packet.Layer(layers.LayerTypeIPv4)
		if lay == nil {
			// fmt.Println("not ipv4")
			continue
		}

		p, ok := lay.(*layers.IPv4)
		if !ok {
			// fmt.Println("not castable")
			continue
		}

		// fmt.Println("got a packet")

		packets += 1
		bytes += int64(len(p.Payload))

		stats, found := statsBySource[p.SrcIP.String()]
		if !found {
			stats = new(statistics)
			statsBySource[p.SrcIP.String()] = stats
		}
		stats.Bytes += int64(len(p.Payload))
		stats.Packets += 1

		now := time.Now()
		if now.Sub(lastUpload) > args.Interval {
			log.Printf("%10d bytes over %10d packets", packets, bytes)

			for k, v := range statsBySource {
				log.Printf("  %15v: %10d bytes over %10d packets", k, v.Bytes, v.Packets)
			}

			// for k, v := range statsByLink {
			// lg.Log(logging.Entry{
			// 	Payload: logEntry{
			// 		From:   k.From,
			// 		To:     k.To,
			// 		Frames: v.Frames,
			// 		Bytes:  v.Bytes,
			// 	},
			// })
			// }

			// log.Printf("uploaded %d log entries\n", len(statsByLink))

			// TODO: delete the entries one-by-one to reduce memory churn
			// statsByLink = make(map[link]*statistics)
			bytes = 0
			packets = 0
			statsBySource = make(map[string]*statistics)
			lastUpload = now
		}

	}

	_ = ctx
	_ = lg
}