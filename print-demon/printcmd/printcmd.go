package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/phin1x/go-ipp"
)

func pdfToPostscript(pdf []byte) ([]byte, error) {
	// create a temporary directory
	tempdir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory for pdf: %w", err)
	}
	defer os.RemoveAll(tempdir)

	tempPdf := filepath.Join(tempdir, "temp.pdf")
	tempPs := filepath.Join(tempdir, "temp.ps")

	// write the pdf to a file
	err = ioutil.WriteFile(tempPdf, pdf, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error writing pdf to temporary file; %w", err)
	}

	// run pdf2ps (TODO: add context)
	stdout, err := exec.Command("pdf2ps", tempPdf, tempPs).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running pdf2s: %w\n%s", err, string(stdout))
	}

	// read the postscript file
	postscript, err := ioutil.ReadFile(tempPs)
	if err != nil {
		return nil, fmt.Errorf("error reading postscript file: %w", err)
	}

	return postscript, nil
}

func Main() error {
	var args struct {
		File    string `arg:"positional,required" help:"File to print"`
		Printer string `arg:"positional,required" help:"URI for printer"`
	}
	arg.MustParse(&args)

	buf, err := os.ReadFile(args.File)
	if err != nil {
		return err
	}

	// define a ipp request
	req := ipp.NewRequest(ipp.OperationPrintJob, 1)
	req.OperationAttributes[ipp.AttributeCharset] = "utf-8"
	req.OperationAttributes[ipp.AttributeNaturalLanguage] = "en"
	req.OperationAttributes[ipp.AttributePrinterURI] = args.Printer
	req.OperationAttributes[ipp.AttributeRequestingUserName] = "some-user"
	req.OperationAttributes[ipp.AttributeDocumentFormat] = "application/octet-stream"

	// encode request to bytes
	payload, err := req.Encode()
	if err != nil {
		return fmt.Errorf("error encoding ipp request: %w", err)
	}
	payload = append(payload, buf...)

	// send ipp request to remote server via http
	httpReq, err := http.NewRequest("POST", args.Printer, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	// set ipp headers
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(payload)))
	httpReq.Header.Set("Content-Type", ipp.ContentTypeIPP)

	// perform the request
	var httpClient http.Client
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error executing http request: %w", err)
	}
	defer resp.Body.Close()

	// read the response
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// response must be 200 for a successful operation
	// other possible http codes are:
	// - 500 -> server error
	// - 426 -> sever requests a encrypted connection
	// - 401 -> forbidden -> need authorization header or user is not permitted
	if resp.StatusCode != 200 {
		return fmt.Errorf("printer said %d: %s", resp.StatusCode, buf)
	}

	// decode ipp response
	msg, err := ipp.NewResponseDecoder(bytes.NewReader(r)).Decode(nil)
	if err != nil {
		return fmt.Errorf("error decoding ipp response: %w", err)
	}

	if err = msg.CheckForErrors(); err != nil {
		return fmt.Errorf("printer responded with error: %v", err)
	}

	// print the response
	fmt.Printf("submitted print job, requestID=%d, status=%d\n", msg.RequestId, msg.StatusCode)
	return nil
}

func main() {
	if err := Main(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
