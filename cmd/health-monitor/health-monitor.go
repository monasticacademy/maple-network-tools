//go:generate protoc -I/usr/local/include -I. --go_out=. healthcheck.proto

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
	"github.com/miekg/dns"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const streamingTraceID = "health-monitor" // identified this client in bigquery debug logs

//go:embed secrets/service-account.json
var googleCredentials []byte

func resolveHost(ctx context.Context, host, nameserver string, out *HealthCheck, wg *sync.WaitGroup) {
	defer wg.Done()

	var msg dns.Msg
	msg.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)

	var client dns.Client
	_, rtt, err := client.ExchangeContext(ctx, &msg, nameserver)

	out.Duration = rtt.Microseconds()
	if err != nil {
		out.Error = err.Error()
	}
}

func pingHost(ctx context.Context, host string, out *HealthCheck, wg *sync.WaitGroup) {
	defer wg.Done()
	rtt, err := pingImpl(ctx, host)
	out.Duration = rtt.Microseconds()
	if err != nil {
		out.Error = err.Error()
	}
}

func pingImpl(ctx context.Context, host string) (time.Duration, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return 0, err
	}
	pinger.SetPrivileged(true)
	pinger.Count = 3

	// it seems that pinger.Run() sometimes hangs forever so we
	// need to respect timeouts from the context
	ch := make(chan error)
	go func() {
		ch <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		<-ch
		return 0, ctx.Err()
	case err = <-ch:
		if err != nil {
			return 0, err
		}
	}

	// collect pinger statistics
	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, err
	}
	return stats.AvgRtt, nil
}

type app struct {
	m           sync.Mutex
	buf         [10][]*HealthCheck            // ring buffer of most recent N entries
	bqClient    *storage.BigQueryWriteClient  // client for writing to bigquery
	descriptor  *descriptorpb.DescriptorProto // the protobuf descriptor for bigquery
	writeStream string                        // the name of the bigquery write stream
}

// push adds a record to the "recent" buffer, possibly dropping old entries
func (a *app) push(r []*HealthCheck) {
	a.m.Lock()
	defer a.m.Unlock()

	copy(a.buf[1:10], a.buf[0:9])
	a.buf[0] = r
}

// recent gets the most recent HealthCheck records, newest is first
func (a *app) latest() [][]*HealthCheck {
	a.m.Lock()
	defer a.m.Unlock()

	// make a copy to avoid data races
	var out [][]*HealthCheck
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

	// set a timeout because the ping function below can hang forever
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	now := time.Now().UnixMicro()

	// run the pings in parallel
	var wg sync.WaitGroup
	var checks []*HealthCheck

	wg.Add(1)
	pingGoogle := HealthCheck{Timestamp: now, Operation: "ping google"}
	go pingHost(ctx, "google.com", &pingGoogle, &wg)
	checks = append(checks, &pingGoogle)

	wg.Add(1)
	pingRouter := HealthCheck{Timestamp: now, Operation: "ping router"}
	go pingHost(ctx, "192.168.88.1", &pingRouter, &wg)
	checks = append(checks, &pingRouter)

	wg.Add(1)
	pingStarlink := HealthCheck{Timestamp: now, Operation: "ping starlink"}
	go pingHost(ctx, "192.168.1.1", &pingStarlink, &wg)
	checks = append(checks, &pingStarlink)

	wg.Add(1)
	resolveAtRouter := HealthCheck{Timestamp: now, Operation: "resolve example.cml using router"}
	go resolveHost(ctx, "example.cml", "192.168.88.1:53", &resolveAtRouter, &wg)
	checks = append(checks, &resolveAtRouter)

	wg.Add(1)
	resolveAtCloudflare := HealthCheck{Timestamp: now, Operation: "resolve example.cml using router"}
	go resolveHost(ctx, "example.cml", "1.1.1.1:53", &resolveAtCloudflare, &wg)
	checks = append(checks, &resolveAtCloudflare)

	wg.Wait()

	// push the result onto the in-memory ring buffer
	a.push(checks)

	// initialize options for protobuf marshalling
	var protoMarshal proto.MarshalOptions

	// serialize the usage data
	var serialized [][]byte
	for _, row := range checks {
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
	args.Table = "HealthCheck"
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
	var x HealthCheck
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
