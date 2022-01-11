package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/pubsub"
	"github.com/alexflint/go-arg"
	"github.com/kr/pretty"
	"github.com/phin1x/go-ipp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
)

// PrintRequest is the json message we recieve from the pubsub topic
type PrintRequest struct {
	Document string `json:"document"` // Document ID for the Google Doc
}

//go:embed secrets/service-account.json
var googleCredentials []byte

// worker contains things that are re-used across many print jobs
type worker struct {
	pubsub  *pubsub.Client
	drive   *drive.Service
	printer string // URI for the physical printer
}

func Main() error {
	ctx := context.Background()

	var args struct {
		Subscription   string `help:"Pubsub queue to pull from"`
		Printer        string `help:"HTTP URI for printer"`
		Document       string `help:"Document ID for the Google Doc"`
		PostscriptFile string `help:"Path to a postscript file"`
	}
	arg.MustParse(&args)

	// unpack google credentials
	creds, err := google.CredentialsFromJSON(ctx,
		googleCredentials,
		"https://www.googleapis.com/auth/documents.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/pubsub",
	)
	if err != nil {
		return fmt.Errorf("error parsing google credentials: %w", err)
	}

	// create the Google pubsub client
	pubsubClient, err := pubsub.NewClient(ctx, creds.ProjectID,
		option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("error creating pubsub client: %w", err)
	}

	// create the Google drive client
	driveClient, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return fmt.Errorf("error creating drive client: %w", err)
	}

	// create a worker
	w := worker{
		pubsub:  pubsubClient,
		drive:   driveClient,
		printer: args.Printer,
	}

	// listen to subscription
	log.Println("listening for messages from pubsub...")
	sub := pubsubClient.Subscription(args.Subscription)
	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		defer func() {
			if ex := recover(); ex != nil {
				log.Println("recovering from panic:", ex)
			}
		}()

		if err := w.processMessage(ctx, m); err != nil {
			log.Println("error processing message: ", err)
		}

		m.Ack()
	})
	if err != nil {
		return fmt.Errorf("error receiving messages from pubsub: %w", err)
	}
	return nil
}

func (w *worker) processMessage(ctx context.Context, m *pubsub.Message) error {
	log.Printf("processing a message from pubsub (%d bytes published %v)",
		len(m.Data), m.PublishTime)

	var job PrintRequest
	err := json.NewDecoder(bytes.NewReader(m.Data)).Decode(&job)
	if err != nil {
		return fmt.Errorf("error decoding pubsub message: %w", err)
	}

	return w.processJob(ctx, &job)
}

func (w *worker) processJob(ctx context.Context, job *PrintRequest) error {
	log.Println("processing job for google doc", job.Document)
	return nil

	// export the document as a zip arcive
	export, err := w.drive.Files.Export(job.Document, "application/pdf").Download()
	if err != nil {
		return fmt.Errorf("error in file download api call: %w", err)
	}
	defer export.Body.Close()

	pdf, err := ioutil.ReadAll(export.Body)
	if err != nil {
		return fmt.Errorf("error reading exported doc from request: %w", err)
	}

	// write the pdf to a file
	fmt.Printf("got %d bytes\n", len(pdf))
	err = ioutil.WriteFile("out.pdf", pdf, os.ModePerm)
	if err != nil {
		return err
	}

	os.Exit(0)

	// define a ipp request
	req := ipp.NewRequest(ipp.OperationPrintJob, 1)
	req.OperationAttributes[ipp.AttributeCharset] = "utf-8"
	req.OperationAttributes[ipp.AttributeNaturalLanguage] = "en"
	req.OperationAttributes[ipp.AttributePrinterURI] = w.printer
	req.OperationAttributes[ipp.AttributeRequestingUserName] = "some-user"
	req.OperationAttributes[ipp.AttributeDocumentFormat] = "application/octet-stream"

	// encode request to bytes
	payload, err := req.Encode()
	if err != nil {
		return fmt.Errorf("error encoding ipp request: %w", err)
	}

	// read the test page
	postscript, err := ioutil.ReadFile("TODO")
	if err != nil {
		return fmt.Errorf("error reading postscript file: %w", err)
	}
	payload = append(payload, postscript...)

	// send ipp request to remote server via http
	httpReq, err := http.NewRequest("POST", w.printer, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	// set ipp headers
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(payload)))
	httpReq.Header.Set("Content-Type", ipp.ContentTypeIPP)

	// perform the request
	var httpClient http.Client
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error executing http request: %w", err)
	}
	defer httpResp.Body.Close()

	// read the response
	buf, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// response must be 200 for a successful operation
	// other possible http codes are:
	// - 500 -> server error
	// - 426 -> sever requests a encrypted connection
	// - 401 -> forbidden -> need authorization header or user is not permitted
	if httpResp.StatusCode != 200 {
		return fmt.Errorf("printer said %d: %s", httpResp.StatusCode, buf)
	}

	// decode ipp response
	resp, err := ipp.NewResponseDecoder(bytes.NewReader(buf)).Decode(nil)
	if err != nil {
		return fmt.Errorf("error decoding ipp response: %w", err)
	}

	// print the response
	fmt.Println("Submitted print job. Response was:")
	pretty.Println(resp)
	return nil
}

func main() {
	err := Main()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
