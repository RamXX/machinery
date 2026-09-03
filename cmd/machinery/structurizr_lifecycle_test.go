package main

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type closeErrorBody struct {
	io.Reader
	err error
}

func (body closeErrorBody) Close() error { return body.err }

func TestDownloadStructurizrArchivePreservesResponseBodyCloseFailure(t *testing.T) {
	previousDo := structurizrHTTPDo
	structurizrHTTPDo = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 injected",
			Body: closeErrorBody{
				Reader: strings.NewReader("failure"),
				err:    errors.New("injected response close failure"),
			},
		}, nil
	}
	t.Cleanup(func() { structurizrHTTPDo = previousDo })

	err := downloadStructurizrArchive("https://invalid.example/structurizr.zip", filepath.Join(t.TempDir(), "archive.zip"))
	if err == nil || !strings.Contains(err.Error(), "HTTP 500 injected") || !strings.Contains(err.Error(), "injected response close failure") {
		t.Fatalf("download did not preserve response and close failures: %v", err)
	}
}

func TestExtractStructurizrZipPreservesReaderCloseFailure(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "structurizr.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("structurizr.sh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "#!/bin/sh\n"); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(writer.Close(), file.Close()); err != nil {
		t.Fatal(err)
	}

	previousClose := closeStructurizrZip
	closeStructurizrZip = func(reader *zip.ReadCloser) error {
		return errors.Join(reader.Close(), errors.New("injected zip close failure"))
	}
	t.Cleanup(func() { closeStructurizrZip = previousClose })

	err = extractStructurizrZip(archive, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "injected zip close failure") {
		t.Fatalf("extraction discarded zip reader close failure: %v", err)
	}
}
