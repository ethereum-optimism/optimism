package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FourByteResponse 4byte.directory API response structure
type FourByteResponse struct {
	Count    int              `json:"count"`
	Next     *string          `json:"next"`
	Previous *string          `json:"previous"`
	Results  []FourByteResult `json:"results"`
}

// FourByteResult represents a single result from the 4byte.directory API
type FourByteResult struct {
	ID             int    `json:"id"`
	CreatedAt      string `json:"created_at"`
	TextSignature  string `json:"text_signature"`
	HexSignature   string `json:"hex_signature"`
	BytesSignature string `json:"bytes_signature"`
}

// FourByteClient is a client for interacting with the 4byte.directory API
type FourByteClient struct {
	client  *http.Client
	url     string
	timeout time.Duration
}

// NewFourByteClient creates a new FourByteClient with a default timeout
func NewFourByteClient() *FourByteClient {
	return &FourByteClient{
		client:  &http.Client{},
		url:     "https://www.4byte.directory/api/v1/signatures/",
		timeout: 10 * time.Second,
	}
}

var DefaultFourByteClient = NewFourByteClient()

// LookupBy4ByteDirectory looks up a 4-byte selector in the 4byte.directory API
func (c *FourByteClient) LookupBy4ByteDirectory(selector string) ([]string, error) {
	// 移除 0x 前缀
	if strings.HasPrefix(selector, "0x") {
		selector = selector[2:]
	}

	// 确保是 8 位十六进制
	if len(selector) < 8 {
		selector = strings.Repeat("0", 8-len(selector)) + selector
	}

	url := fmt.Sprintf("%s?hex_signature=0x%s", c.url, selector)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "cast4byte-go/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response FourByteResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var signatures []string
	for _, result := range response.Results {
		signatures = append(signatures, result.TextSignature)
	}

	return signatures, nil
}
