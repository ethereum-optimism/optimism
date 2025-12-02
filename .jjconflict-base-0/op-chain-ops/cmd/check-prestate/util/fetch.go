package util

import (
	"fmt"
	"io"
	"net/http"
)

func Fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %v: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %v: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
