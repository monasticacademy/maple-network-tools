//go:generate protoc -I/usr/local/include -I. --go_out=. traffic.proto

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"time"

	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/kr/pretty"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"

	storage "cloud.google.com/go/bigquery/storage/apiv1beta2"
	storagepb "google.golang.org/genproto/googleapis/cloud/bigquery/storage/v1beta2"
)

//go:embed service-account.json
var googleCredentials []byte

// this is how we match 802.11 packet addresses to the ARP table
func arpKey(hw net.HardwareAddr) string {
	if len(hw) != 6 {
		return "nil"
	}
	return fmt.Sprintf("%X:%X", hw[4], hw[5])
}

func dot11Key(hw net.HardwareAddr) string {
	if len(hw) != 6 {
		return "nil"
	}
	return fmt.Sprintf("%X:%X", hw[0], hw[1])
}

// trafficRow is a row in the bigquery table for traffic summaries
type trafficRow struct {
	Begin     time.Time // beginning of the time period when these packets were collected
	Duration  int64     // length in milliseconds of period when these packets were collected
	IPAddress string    // address of computer
	Bytes     int64     // number of bytes observed
	Packets   int64     // number of packets observed
}

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
		Dataset   string
		Table     string
		Interval  time.Duration
	}
	args.LogName = "orbi-packets"
	args.Dataset = "maple"
	args.Table = "router_traffic"
	args.Interval = 10 * time.Second
	arg.MustParse(&args)

	// unpack google credentials
	creds, err := google.CredentialsFromJSON(ctx, googleCredentials)
	if err != nil {
		log.Fatal("error parsing credentials: ", err)
	}

	log.Println("project:", creds.ProjectID)
	log.Println("dataset:", args.Dataset)
	log.Println("table:", args.Table)
	log.Println("log interval:", args.Interval)

	// create the logger
	logClient, err := logging.NewClient(ctx,
		creds.ProjectID,
		option.WithCredentialsJSON(googleCredentials))
	if err != nil {
		log.Fatal("error creating logging client: ", err)
	}
	defer logClient.Close()

	lg := logClient.Logger(args.LogName)
	_ = lg

	// create the bigquery client for stream insertion
	bqClient, err := storage.NewBigQueryWriteClient(ctx,
		option.WithCredentialsJSON(googleCredentials))
	if err != nil {
		log.Fatal(err)
	}
	defer bqClient.Close()

	// create the write stream
	parent := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", creds.ProjectID, args.Dataset, args.Table)
	resp, err := bqClient.CreateWriteStream(ctx, &storagepb.CreateWriteStreamRequest{
		Parent: parent,
		WriteStream: &storagepb.WriteStream{
			Type: storagepb.WriteStream_COMMITTED,
		},
	})
	if err != nil {
		log.Fatal("CreateWriteStream: ", err)
	}

	// get the stream
	stream, err := bqClient.AppendRows(ctx)
	if err != nil {
		log.Fatal("AppendRows: ", err)
	}

	row := Traffic{
		Begin:     "foo",
		Duration:  3,
		IPAddress: "10.0.0.0",
		Bytes:     4,
		Packets:   5,
	}

	bs, err := proto.MarshalOptions{}.Marshal(&row)
	if err != nil {
		log.Fatal("protobuf.Marshal: ", err)
	}

	// eep protobuf is crazy
	descriptor, err := adapt.NormalizeDescriptor(row.ProtoReflect().Descriptor())
	if err != nil {
		log.Fatal("NormalizeDescriptor: ", err)
	}

	// append the rows
	err = stream.Send(&storagepb.AppendRowsRequest{
		WriteStream: resp.Name,
		TraceId:     "orbi-monitor", // identifies this client
		Rows: &storagepb.AppendRowsRequest_ProtoRows{
			ProtoRows: &storagepb.AppendRowsRequest_ProtoData{
				// protocol buffer schema
				WriterSchema: &storagepb.ProtoSchema{
					ProtoDescriptor: descriptor,
				},
				// protobuf-encoded data
				Rows: &storagepb.ProtoRows{
					SerializedRows: [][]byte{
						bs,
					}, // serialized protocol buffer data
				},
			},
		},
	})
	if err != nil {
		log.Fatal("AppendRows.Send: ", err)
	}

	aresp, err := stream.Recv()
	if err != nil {
		log.Fatal("AppendRows.Recv: ", err)
	}
	pretty.Println("AppendRows response was: ", aresp.GetResponse())

	fresp, err := bqClient.FinalizeWriteStream(ctx, &storagepb.FinalizeWriteStreamRequest{
		Name: resp.Name,
	})
	if err != nil {
		log.Fatal("FinalizeWriteStream: ", err)
	}

	err = bqClient.Close()
	if err != nil {
		log.Fatal("BigQueryWriteClient.Close: ", err)
	}

	log.Printf("sent %d bytes and bigquery says it added %d rows", len(bs), fresp.RowCount)
	return

	// open ARP table (TODO: should reload this periodically)
	arpTable, err := loadARPTable()
	if err != nil {
		log.Fatal("error loading ARP table: ", err)
	}
	ipByMAC := make(map[string]net.IP)
	for _, entry := range arpTable.entries {
		ipByMAC[arpKey(entry.HardwareAddr)] = entry.IPAddr
		log.Printf("  %20v %20v", entry.HardwareAddr, entry.IPAddr)
	}
	log.Printf("loaded %d MAC addresses from ARP table", len(ipByMAC))

	// open packet capture handle
	handle, err := pcapgo.NewEthernetHandle(args.Interface)
	if err != nil {
		log.Fatal("error creating ethernet handle: ", err)
	}

	statsBySource := make(map[string]*statistics)
	// statsByLink := make(map[link]*statistics)

	logTicker := time.NewTicker(args.Interval)
	var lastDump time.Time

	var packets, bytes, nonwifi int64
	pkgsrc := gopacket.NewPacketSource(handle, layers.LayerTypeDot11)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer

		case <-logTicker.C:
			log.Println()
			log.Printf("%10d bytes over %10d packets (plus %d non-wifi)", packets, bytes, nonwifi)
			for k, v := range statsBySource {
				//k = strings.ToUpper(k)
				log.Printf("  %15v %10d bytes over %10d packets", k, v.Bytes, v.Packets)
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

		case packet := <-pkgsrc.Packets():

			var dump bool
			if time.Since(lastDump) > 20*time.Second {
				dump = true
			}

			if dump {
				fmt.Println("packet:")
				for _, lay := range packet.Layers() {
					fmt.Println("  ", lay.LayerType())
				}
				lastDump = time.Now()
			}

			lay := packet.Layer(layers.LayerTypeDot11)
			if lay == nil {
				nonwifi += 1
				break
			}

			p, ok := lay.(*layers.Dot11)
			if !ok {
				nonwifi += 1
				break
			}

			// if dump {
			// 	fmt.Printf("  %v -> %v (%v, %v)\n",
			// 		p.Address1,
			// 		p.Address2,
			// 		p.Address3,
			// 		p.Address4)
			// 	fmt.Printf("  type: %v\n", p.Type)
			// }

			// for _, x := range packet.Layers() {
			// 	if x.LayerType() == layers.LayerTypeDot11InformationElement {
			// 		if xx, ok := x.(*layers.Dot11InformationElement); ok {
			// 			fmt.Printf("information element: %25v (%d bytes)\n", xx.ID, len(xx.Info))
			// 		}
			// 	}
			// }

			macs := []net.HardwareAddr{p.Address1, p.Address2, p.Address3, p.Address4}
			for _, mac := range macs {
				if len(mac) != 6 {
					continue
				}

				var ip net.IP
				var found bool
				if ip, found = ipByMAC[dot11Key(mac)]; !found {
					continue
				}
				//log.Printf("known hardware address found at position %d (%v)", i, ip)

				stats, found := statsBySource[ip.String()]
				if !found {
					stats = new(statistics)
					statsBySource[ip.String()] = stats
				}
				stats.Bytes += int64(len(p.Payload))
				stats.Packets += 1
			}

			packets += 1
			bytes += int64(len(p.Payload))

		}
	}
}
