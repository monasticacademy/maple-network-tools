package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// Convert PDF to PCL using ghostscript
func pdfToPCL(ctx context.Context, pdf []byte) ([]byte, error) {
	// create a temporary directory
	tempdir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory for pdf: %w", err)
	}
	defer os.RemoveAll(tempdir)

	tempPdf := filepath.Join(tempdir, "temp.pdf")
	tempPcl := filepath.Join(tempdir, "temp.pcl")

	// write the pdf to a file
	err = ioutil.WriteFile(tempPdf, pdf, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error writing pdf to temporary file; %w", err)
	}

	// run ghostscript
	stdout, err := exec.CommandContext(
		ctx,
		"gs",
		"-o", tempPcl,
		"-sDEVICE=pxlcolor",
		"-f", tempPdf).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running pdf2s: %w\n%s", err, string(stdout))
	}

	// read the converted file
	pcl, err := ioutil.ReadFile(tempPcl)
	if err != nil {
		return nil, fmt.Errorf("error reading PCL file: %w", err)
	}

	return pcl, nil
}

// Convert PDF to postscript using pdf2ps
func pdfToPostscript(ctx context.Context, pdf []byte) ([]byte, error) {
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
	stdout, err := exec.CommandContext(ctx, "pdf2ps", tempPdf, tempPs).CombinedOutput()
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
