//go:generate protoc -I/usr/local/include -I. --go_out=. reachability.proto

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"sync"
	"time"

	storage "cloud.google.com/go/bigquery/storage/apiv1beta2"
	storagepb "google.golang.org/genproto/googleapis/cloud/bigquery/storage/v1beta2"

	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"github.com/alexflint/go-arg"
	"github.com/go-ping/ping"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const streamingTraceID = "uplink-monitor" // identified this client in bigquery debug logs

//go:embed secrets/service-account.json
var googleCredentials []byte

func pingHost(ctx context.Context, host string, success *bool, info *string, latency *int64) {
	t, err := pingImpl(ctx, host)
	if err == nil {
		*success = true
		*info = ""
		*latency = t.Microseconds()
		log.Printf("ping %s -> success (%v)", host, t)
	} else {
		*success = false
		*info = err.Error()
		*latency = 0
		log.Printf("ping %s -> fail (%v)", host, err)
	}
}

func pingImpl(ctx context.Context, host string) (time.Duration, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return 0, err
	}
	pinger.SetPrivileged(true)
	pinger.Count = 3
	err = pinger.Run() // blocks until finished
	if err != nil {
		return 0, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, err
	}
	return stats.AvgRtt, nil
}

type app struct {
	m           sync.Mutex
	buf         [10]*Reachability             // ring buffer of most recent N entries
	bqClient    *storage.BigQueryWriteClient  // client for writing to bigquery
	descriptor  *descriptorpb.DescriptorProto // the protobuf descriptor for bigquery
	writeStream string                        // the name of the bigquery write stream
}

// push adds a record to the "recent" buffer, possibly dropping old entries
func (a *app) push(r *Reachability) {
	a.m.Lock()
	defer a.m.Unlock()

	copy(a.buf[1:10], a.buf[0:9])
	a.buf[0] = r
}

// recent gets the most recent reachability records, newest is first
func (a *app) latest() []*Reachability {
	a.m.Lock()
	defer a.m.Unlock()

	// make a copy to avoid data races
	var out []*Reachability
	for _, r := range a.buf {
		if r == nil {
			break
		}
		out = append(out, r)
	}
	return out
}

// tick gets executed every 1 minute. It pings the modem, router, and google.com.
func (a *app) tick(ctx context.Context) error {
	timestamp := time.Now()
	log.Println("tick")

	var r Reachability
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		pingHost(ctx, "google.com", &r.GoogleReachable, &r.GoogleError, &r.GoogleLatency)
		wg.Done()
	}()
	go func() {
		pingHost(ctx, "microtik.maple.cml.me", &r.RouterReachable, &r.RouterError, &r.RouterLatency)
		wg.Done()
	}()
	go func() {
		pingHost(ctx, "ridgewave.maple.cml.me", &r.ModemReachable, &r.ModemError, &r.ModemLatency)
		wg.Done()
	}()
	wg.Wait() // TODO: set a timeout on this

	a.push(&r)

	// initialize options for protobuf marshalling
	var protoMarshal proto.MarshalOptions

	// serialize the usage data
	var serialized [][]byte
	for _, row := range []*Reachability{&r} {
		row.Timestamp = timestamp.UnixMicro()
		buf, err := protoMarshal.Marshal(row)
		if err != nil {
			return fmt.Errorf("protobuf.Marshal: %w", err)
		}
		serialized = append(serialized, buf)
	}

	// get the stream for pushing data to bigquery
	bqStream, err := a.bqClient.AppendRows(ctx)
	if err != nil {
		return fmt.Errorf("AppendRows: %w", err)
	}

	// push the data to bigquery
	err = bqStream.Send(&storagepb.AppendRowsRequest{
		WriteStream: a.writeStream,
		TraceId:     streamingTraceID, // identifies this client
		Rows: &storagepb.AppendRowsRequest_ProtoRows{
			ProtoRows: &storagepb.AppendRowsRequest_ProtoData{
				WriterSchema: &storagepb.ProtoSchema{
					ProtoDescriptor: a.descriptor,
				},
				Rows: &storagepb.ProtoRows{
					SerializedRows: serialized,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error in stream.Send: %w", err)
	}

	// get the response
	_, err = bqStream.Recv()
	if err != nil {
		return fmt.Errorf("error in stream.Recv: %w", err)
	}

	// this can help to diagnose errors
	//pretty.Println("AppendRows response was: ", resp.GetResponse())

	log.Printf("sent %d rows to bigquery", len(serialized))
	return nil
}

func main() {
	ctx := context.Background()

	var args struct {
		Port     string `http:"Port for the HTTP user interface"`
		Dataset  string `help:"Bigquery dataset name"`
		Table    string `help:"Bigquery table name"`
		Interval time.Duration
	}
	args.Port = ":8000"
	args.Dataset = "network"
	args.Table = "reachability"
	args.Interval = time.Minute
	arg.MustParse(&args)

	// unpack google credentials
	creds, err := google.CredentialsFromJSON(ctx, googleCredentials)
	if err != nil {
		log.Fatal("error parsing credentials: ", err)
	}

	log.Println("project:", creds.ProjectID)
	log.Println("dataset:", args.Dataset)
	log.Println("table:", args.Table)
	log.Println("interval:", args.Interval)

	// create the bigquery client for stream insertion
	bqClient, err := storage.NewBigQueryWriteClient(ctx,
		option.WithCredentialsJSON(googleCredentials))
	if err != nil {
		log.Fatal(err)
	}
	defer bqClient.Close()

	// create the bigquery write stream
	parent := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", creds.ProjectID, args.Dataset, args.Table)
	resp, err := bqClient.CreateWriteStream(ctx, &storagepb.CreateWriteStreamRequest{
		Parent: parent,
		WriteStream: &storagepb.WriteStream{
			Type: storagepb.WriteStream_COMMITTED,
		},
	})
	if err != nil {
		log.Fatal("error creating write stream: ", err)
	}

	// get descriptor for our protobuf representing a bigquery row
	var x Reachability
	descriptor, err := adapt.NormalizeDescriptor(x.ProtoReflect().Descriptor())
	if err != nil {
		log.Fatal("error normalizing protobuf descriptor: ", err)
	}

	app := app{
		bqClient:    bqClient,
		descriptor:  descriptor,
		writeStream: resp.Name,
	}

	// start the web UI
	go app.runWebUI(ctx, args.Port)

	// run the tick function once
	err = app.tick(ctx)
	if err != nil {
		log.Println(err)
	}

	// grab traffic snapshots and send to bigquery every N minutes
	ticker := time.NewTicker(args.Interval)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer

		case <-ticker.C:
			err = app.tick(ctx)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
