package httputil

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

type Downloader struct {
	Client     *http.Client
	Progressor ioutil.Progressor
	MaxSize    int64
}

func (d *Downloader) Download(ctx context.Context, url string, out io.Writer) error {
	if out == nil {
		return fmt.Errorf("output writer is nil")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with status code %d: %s", resp.StatusCode, resp.Status)
	}
	if resp.ContentLength > 0 && d.MaxSize > 0 && resp.ContentLength > d.MaxSize {
		return fmt.Errorf("content length %d exceeds maximum allowed size %d", resp.ContentLength, d.MaxSize)
	}

	defer resp.Body.Close()

	r := io.Reader(resp.Body)
	if d.MaxSize > 0 {
		r = io.LimitReader(resp.Body, d.MaxSize)
	}

	pr := &ioutil.ProgressReader{
		R:          r,
		Progressor: d.Progressor,
		Total:      resp.ContentLength,
	}
	if _, err := io.Copy(out, pr); err != nil {
		return fmt.Errorf("failed to write download: %w", err)
	}
	return nil
}
