//go:generate protoc -I/usr/local/include -I. --go_out=. usage.proto

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	storage "cloud.google.com/go/bigquery/storage/apiv1beta2"
	storagepb "google.golang.org/genproto/googleapis/cloud/bigquery/storage/v1beta2"

	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
	"github.com/dustin/go-humanize"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

const streamingTraceID = "microtik-traffic" // identified this client in bigquery debug logs

type DHCPLease struct {
	IP       string
	MAC      string
	Hostname string
}

type Traffic struct {
	From    string
	To      string
	Packets int
	Bytes   int
}

// get the last number in an IP address
func lastPart(ip string) string {
	if n := strings.LastIndex(ip, "."); n >= 0 {
		return ip[n+1:]
	}
	return ip
}

// checks whether token starts with the given prefix and returns the remainder of the string
func hasPrefix(token, key string) (value string, ok bool) {
	if strings.HasPrefix(token, key) {
		return strings.TrimPrefix(token, key), true
	}
	return "", false
}

//go:embed microtik.pub
var microtikServerKey []byte

//go:embed gcloud-service-account.json
var googleCredentials []byte

func main() {
	ctx := context.Background()

	var args struct {
		LogName  string `help:"String for cloud logging"`
		Dataset  string `help:"Bigquery dataset name"`
		Table    string `help:"Bigquery table name"`
		Router   string `help:"Hostname or IP address of router"`
		User     string `help:"SSH username for router"`
		Pass     string `help:"SSH password for router" arg:"env:PASS"`
		Interval time.Duration
	}
	args.LogName = "microtik-traffic"
	args.Dataset = "maple"
	args.Table = "router_usage"
	args.Interval = 10 * time.Minute
	args.Router = "router.lan"
	args.User = "traffic-monitor"
	arg.MustParse(&args)

	// parse the embeded public key for our router
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(microtikServerKey)
	if err != nil {
		log.Fatal("error parsing server SSH key: ", err)
	}

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

	// initialize options for protobuf marshalling
	var protoMarshal proto.MarshalOptions

	// get descriptor for our protobuf representing a bigquery row
	var usage Usage
	trafficDescriptor, err := adapt.NormalizeDescriptor(usage.ProtoReflect().Descriptor())
	if err != nil {
		log.Fatal("NormalizeDescriptor: ", err)
	}

	// clear the microtik snapshot table so that we don't get a spike at the start
	// open an ssh connection to the router
	log.Println("clearing the router snapshot table")
	sshClient, err := ssh.Dial("tcp", args.Router, &ssh.ClientConfig{
		User:            args.User,
		Auth:            []ssh.AuthMethod{ssh.Password(args.Pass)},
		HostKeyCallback: ssh.FixedHostKey(pubkey),
		Timeout:         3 * time.Second,
	})
	if err != nil {
		log.Fatal("error dialing router for ssh: ", err)
	}

	// create a session to run the command
	snapshotSession, err := sshClient.NewSession()
	if err != nil {
		log.Fatal("error opening SSH session: ", err)
	}
	defer snapshotSession.Close()

	// take a snapshot
	err = snapshotSession.Run("/ip accounting snapshot take")
	if err != nil {
		log.Fatal("error taking traffic snapshot: ", err)
	}

	// the following function executes every N minutes
	tick := func(ctx context.Context) error {
		// open an ssh connection to the router
		// do not re-use across ticks because it will time out
		sshClient, err := ssh.Dial("tcp", args.Router, &ssh.ClientConfig{
			User:            args.User,
			Auth:            []ssh.AuthMethod{ssh.Password(args.Pass)},
			HostKeyCallback: ssh.FixedHostKey(pubkey),
			Timeout:         3 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("error dialing router for ssh: %w", err)
		}

		// create a session to fetch the dhcp lease table
		dhcpSession, err := sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("error opening SSH session: %w", err)
		}
		defer dhcpSession.Close()

		// fetch the DHCP lease table
		dhcpBuf, err := dhcpSession.CombinedOutput("/ip dhcp-server lease print terse")
		if err != nil {
			return fmt.Errorf("error running DHCP command: %w", err)
		}

		// parse the dhcp lease table
		var leases []DHCPLease
		for _, line := range strings.Split(string(dhcpBuf), "\n") {
			var item DHCPLease
			for _, tok := range strings.Split(line, " ") {
				if v, ok := hasPrefix(tok, "address="); ok {
					item.IP = v
				}
				if v, ok := hasPrefix(tok, "mac-address="); ok {
					item.MAC = v
				}
				if v, ok := hasPrefix(tok, "host-name="); ok {
					item.Hostname = v
				}
			}
			var zero DHCPLease
			if item != zero {
				leases = append(leases, item)
			}
		}

		// create a session to take the traffic snapshot
		snapshotSession, err := sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("error opening SSH session: %w", err)
		}
		defer snapshotSession.Close()

		// fetch the traffic table
		err = snapshotSession.Run("/ip accounting snapshot take")
		if err != nil {
			return fmt.Errorf("error taking traffic snapshot: %w", err)
		}

		// create a session to fetch the traffic table
		trafficSession, err := sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("error opening SSH session: %w", err)
		}
		defer trafficSession.Close()

		// fetch the traffic table
		trafficBuf, err := trafficSession.CombinedOutput("/ip accounting snapshot print terse")
		if err != nil {
			return fmt.Errorf("error printing traffic snapshot: %w", err)
		}

		// parse the traffic table
		var traffic []Traffic
		for _, line := range strings.Split(string(trafficBuf), "\n") {
			var item Traffic
			for _, tok := range strings.Split(line, " ") {
				if v, ok := hasPrefix(tok, "src-address="); ok {
					item.From = v
				}
				if v, ok := hasPrefix(tok, "dst-address="); ok {
					item.To = v
				}
				if v, ok := hasPrefix(tok, "packets="); ok {
					item.Packets, err = strconv.Atoi(v)
					if err != nil {
						log.Println(err)
					}
				}
				if v, ok := hasPrefix(tok, "bytes="); ok {
					item.Bytes, err = strconv.Atoi(v)
					if err != nil {
						log.Println(err)
					}
				}
			}
			var zero Traffic
			if item != zero {
				traffic = append(traffic, item)
			}
		}

		// calculate usage per hostname
		hostnameByIP := make(map[string]string)
		for _, lease := range leases {
			hostnameByIP[lease.IP] = lease.Hostname
		}

		usageByHostname := make(map[string]*Usage)
		for _, row := range traffic {
			var localIP string
			if strings.HasPrefix(row.From, "192.168.88.") {
				localIP = row.From
			} else if strings.HasPrefix(row.To, "192.168.88.") {
				localIP = row.To
			} else {
				continue
			}

			hostname, ok := hostnameByIP[localIP]
			if !ok {
				hostname = "unknown." + lastPart(localIP)
			}
			if hostname == "" {
				hostname = "unnamed." + lastPart(localIP)
			}

			usage := usageByHostname[hostname]
			if usage == nil {
				usage = &Usage{Host: hostname}
				usageByHostname[hostname] = usage
			}
			usage.Bytes += int64(row.Bytes)
			usage.Packets += int64(row.Packets)
		}

		// print usage info
		var usages []*Usage
		for _, usage := range usageByHostname {
			usages = append(usages, usage)
		}
		sort.Slice(usages, func(i, j int) bool {
			return usages[i].Bytes > usages[j].Bytes
		})
		for _, usage := range usages {
			log.Printf("%40s %15s %10d packets\n",
				usage.Host, humanize.Bytes(uint64(usage.Bytes)), usage.Packets)
		}

		// serialize the usage data
		now := time.Now()
		var bqRows, bqBytes int
		var rowData [][]byte
		for _, usage := range usageByHostname {
			usage.Begin = now.String()
			usage.Duration = args.Interval.Milliseconds()

			buf, err := protoMarshal.Marshal(usage)
			if err != nil {
				return fmt.Errorf("protobuf.Marshal: %w", err)
			}

			bqRows += 1
			bqBytes += len(buf)
			rowData = append(rowData, buf)
		}

		// get the stream for pushing data to bigquery
		bqStream, err := bqClient.AppendRows(ctx)
		if err != nil {
			log.Fatal("AppendRows: ", err)
		}

		// push the data to bigquery
		err = bqStream.Send(&storagepb.AppendRowsRequest{
			WriteStream: resp.Name,
			TraceId:     streamingTraceID, // identifies this client
			Rows: &storagepb.AppendRowsRequest_ProtoRows{
				ProtoRows: &storagepb.AppendRowsRequest_ProtoData{
					// protocol buffer schema
					WriterSchema: &storagepb.ProtoSchema{
						ProtoDescriptor: trafficDescriptor,
					},
					// protocol buffer data
					Rows: &storagepb.ProtoRows{
						SerializedRows: rowData, // serialized protocol buffer data
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("AppendRows.Send: %w", err)
		}

		// get the response
		_, err = bqStream.Recv()
		if err != nil {
			return fmt.Errorf("AppendRows.Recv: %w", err)
		}

		// this can help to diagnose errors
		//pretty.Println("AppendRows response was: ", resp.GetResponse())

		log.Printf("sent %d rows (%d bytes) to bigquery", bqRows, bqBytes)
		return nil
	}

	// grab traffic snapshots and send to bigquery every N minutes
	ticker := time.NewTicker(args.Interval)

outer:
	for {
		select {
		case <-ctx.Done():
			break outer

		case <-ticker.C:
			err = tick(ctx)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
