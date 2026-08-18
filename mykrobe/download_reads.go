package mykrobe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

const TBTestReadsURL = "https://ndownloader.figshare.com/files/21059229"

// DownloadTBTestReads downloads the reads used to test the TB panels.
func DownloadTBTestReads(ctx context.Context, outputPath string) error {
	return downloadFile(ctx, http.DefaultClient, TBTestReadsURL, outputPath)
}

func downloadFile(ctx context.Context, client *http.Client, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create reads download request: %w", err)
	}
	out, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", outputPath, err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(outputPath)
		}
	}()
	resp, err := client.Do(req)
	if err != nil {
		_ = out.Close()
		return fmt.Errorf("download reads: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = out.Close()
		return fmt.Errorf("download reads: %s", resp.Status)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return fmt.Errorf("write output file %q: %w", outputPath, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close output file %q: %w", outputPath, err)
	}
	complete = true
	return nil
}
