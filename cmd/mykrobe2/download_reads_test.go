package main

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadTestReadsRequiresOutputFilename(t *testing.T) {
	cmd := newDownloadTestReadsCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Fatalf("error = %v, want required output filename error", err)
	}
}

func TestDownloadTestReadsRoutesOutputFilename(t *testing.T) {
	previous := downloadTBTestReads
	t.Cleanup(func() {
		downloadTBTestReads = previous
	})

	var gotOutput string
	downloadTBTestReads = func(ctx context.Context, outputPath string) error {
		gotOutput = outputPath
		return nil
	}

	output := filepath.Join(t.TempDir(), "reads.fq.gz")
	cmd := newDownloadTestReadsCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{output})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if gotOutput != output {
		t.Fatalf("output = %q, want %q", gotOutput, output)
	}
	if !strings.Contains(stdout.String(), output) {
		t.Fatalf("stdout = %q, want output filename", stdout.String())
	}
}
