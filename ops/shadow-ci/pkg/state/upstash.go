package state

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// UpstashStore implements Store using Upstash Redis REST API.
// Free tier: 10K commands/day, 256MB. No client library needed.
//
// Env vars: UPSTASH_REDIS_REST_URL, UPSTASH_REDIS_REST_TOKEN
type UpstashStore struct {
	client *http.Client
	url    string // e.g. "https://xyz.upstash.io"
	token  string
	prefix string // key prefix, e.g. "shadow-ci:"
}

// NewUpstashStore creates an Upstash Redis-backed state store.
func NewUpstashStore(url, token, prefix string) *UpstashStore {
	if prefix == "" {
		prefix = "shadow-ci:"
	}
	return &UpstashStore{
		client: &http.Client{},
		url:    strings.TrimRight(url, "/"),
		token:  token,
		prefix: prefix,
	}
}

func (s *UpstashStore) Load(key string) ([]byte, error) {
	resp, err := s.do("GET", s.prefix+key)
	if err != nil {
		return nil, err
	}

	// Upstash returns {"result": <value>} where value is null if not found.
	var result struct {
		Result *string `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing upstash response: %w", err)
	}
	if result.Result == nil {
		return nil, ErrNotFound
	}

	// Values are stored base64-encoded to handle binary JSON safely.
	return base64.StdEncoding.DecodeString(*result.Result)
}

func (s *UpstashStore) Save(key string, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	_, err := s.do("SET", s.prefix+key, encoded)
	return err
}

// do executes an Upstash REST command. Args are command parts: "GET", "mykey" or "SET", "mykey", "value".
func (s *UpstashStore) do(args ...string) ([]byte, error) {
	// Upstash REST API: POST <url>/ with JSON body ["CMD", "arg1", "arg2", ...]
	body, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.url+"/", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstash request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstash %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
