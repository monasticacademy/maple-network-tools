package main

import (
	"context"
	_ "embed"
	"log"
	"time"

	"cloud.google.com/go/logging"
	"github.com/alexflint/go-arg"
	"github.com/showwin/speedtest-go/speedtest"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

//go:embed secrets/service-account.json
var googleCredentials []byte

// logEntry is the data that send to cloud logging once per N seconds
type logEntry struct {
	Machine  string  // machine from which this test was peformed (e.g. "chromebook-1")
	Location string  // physical location from which this test was performed (e.g. "garuda")
	Network  string  // the wifi or ethernet network from which this test was performed (e.g. "maple-wifi")
	Carrier  string  // the carrier over which this test was performed (e.g. "vtel")
	Latency  int64   // ping time in milliseconds
	Down     float64 // download speed in megabits per second
	Up       float64 // upload speed in megabits per second
}

func main() {
	ctx := context.Background()

	// process command line arguments
	var args struct {
		Machine  string `arg:"required,env:SPEEDLOGGER_MACHINE"`
		Location string `arg:"required,env:SPEEDLOGGER_LOCATION"`
		Network  string `arg:"required,env:SPEEDLOGGER_NETWORK"`
		Carrier  string `arg:"required,env:SPEEDLOGGER_CARRIER"`
		LogName  string
		Interval time.Duration
		Timeout  time.Duration `help:"timeout for speed test"`
	}
	args.LogName = "internet-speed"
	args.Interval = 30 * time.Minute
	args.Timeout = 5 * time.Minute
	arg.MustParse(&args)

	// unpack google credentials
	creds, err := google.CredentialsFromJSON(ctx, googleCredentials)
	if err != nil {
		log.Fatal("error parsing credentials: ", err)
	}

	log.Println("machine:", args.Machine)
	log.Println("location:", args.Location)
	log.Println("netwowrk:", args.Network)
	log.Println("carrier:", args.Carrier)
	log.Println("project:", creds.ProjectID)
	log.Println("log interval:", args.Interval)

	// set up the logger
	logClient, err := logging.NewClient(ctx, creds.ProjectID,
		option.WithCredentialsJSON(googleCredentials))
	if err != nil {
		log.Fatal("error creating logging client: ", err)
	}
	defer logClient.Close()

	lg := logClient.Logger(args.LogName)

	// set up the speed test
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		log.Fatal(err)
	}

	serverList, err := speedtest.FetchServerList(user)
	if err != nil {
		log.Fatal(err)
	}
	if len(serverList.Servers) == 0 {
		log.Fatal("speedtest found zero servers")
	}

	// the servers are returned to us sorted by distance, so use the first one
	s := serverList.Servers[0]

	// create the ticker to run speed tests
	ticker := time.NewTicker(args.Interval)
	defer ticker.Stop()

	// the following loop ticks immediately, then waits for the intervals
	for ; true; <-ticker.C {
		var entry logEntry
		entry.Machine = args.Machine
		entry.Location = args.Location
		entry.Network = args.Network
		entry.Carrier = args.Carrier

		log.Println("running a speed test...")

		// do the ping test
		pingCtx, _ := context.WithTimeout(ctx, args.Timeout)
		err = s.PingTestContext(pingCtx)
		if err == nil {
			entry.Latency = s.Latency.Milliseconds()
		} else {
			log.Println("ping test failed: ", err)
			lg.StandardLogger(logging.Error).Println("ping test failed: ", err)
		}

		// do the download test
		downloadCtx, _ := context.WithTimeout(ctx, args.Timeout)
		err = s.DownloadTestContext(downloadCtx, false)
		if err == nil {
			entry.Down = s.DLSpeed
		} else {
			log.Println("download test failed: ", err)
			lg.StandardLogger(logging.Error).Println("download test failed: ", err)
		}

		// do the upload test
		uploadCtx, _ := context.WithTimeout(ctx, args.Timeout)
		err = s.UploadTestContext(uploadCtx, false)
		if err == nil {
			entry.Up = s.ULSpeed
		} else {
			log.Println("upload test failed: ", err)
			lg.StandardLogger(logging.Error).Println("upload test failed: ", err)
		}

		log.Printf("%#v\n", entry)
		lg.Log(logging.Entry{Payload: entry})
	}
}
