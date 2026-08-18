package mykrobe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadFile(t *testing.T) {
	const reads = "@read1\nACGT\n+\nIIII\n"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
		if req.URL.String() != "https://example.test/reads.fq.gz" {
			t.Errorf("URL = %s", req.URL)
		}
		return response(http.StatusOK, io.NopCloser(strings.NewReader(reads))), nil
	})}

	output := filepath.Join(t.TempDir(), "reads.fq.gz")
	if err := downloadFile(context.Background(), client, "https://example.test/reads.fq.gz", output); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != reads {
		t.Fatalf("output = %q, want %q", got, reads)
	}
}

func TestDownloadFileRejectsExistingOutputBeforeRequest(t *testing.T) {
	requested := false
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requested = true
		return response(http.StatusOK, io.NopCloser(strings.NewReader("replacement"))), nil
	})}
	output := filepath.Join(t.TempDir(), "reads.fq.gz")
	if err := os.WriteFile(output, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := downloadFile(context.Background(), client, "https://example.test/reads.fq.gz", output)
	if err == nil || !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("error = %v, want file exists error", err)
	}
	if requested {
		t.Fatal("HTTP request made for an existing output file")
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("existing output changed to %q", got)
	}
}

func TestDownloadFileHTTPErrorDoesNotCreateOutput(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(http.StatusNotFound, io.NopCloser(strings.NewReader("not found"))), nil
	})}
	output := filepath.Join(t.TempDir(), "reads.fq.gz")
	err := downloadFile(context.Background(), client, "https://example.test/reads.fq.gz", output)
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("error = %v, want HTTP status error", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("partial output exists after HTTP error: %v", err)
	}
}

func TestDownloadFileRemovesPartialOutput(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(io.MultiReader(strings.NewReader("partial"), errorReader{}))
		return response(http.StatusOK, body), nil
	})}
	output := filepath.Join(t.TempDir(), "reads.fq.gz")
	if err := downloadFile(context.Background(), client, "https://example.test/reads.fq.gz", output); err == nil {
		t.Fatal("expected interrupted response error")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("partial output exists after write error: %v", err)
	}
}

func response(statusCode int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     strconv.Itoa(statusCode) + " " + http.StatusText(statusCode),
		Body:       body,
		Header:     make(http.Header),
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("interrupted download")
}
