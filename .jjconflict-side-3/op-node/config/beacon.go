package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type L1BeaconEndpointSetup interface {
	Setup(ctx context.Context, log log.Logger) (cl apis.BeaconClient, fb []apis.BlobSideCarsClient, err error)
	// ShouldIgnoreBeaconCheck returns true if the Beacon-node version check should not halt startup.
	ShouldIgnoreBeaconCheck() bool
	ShouldFetchAllSidecars() bool
	Check() error
}

type L1BeaconEndpointConfig struct {
	BeaconAddr             string   // Address of L1 User Beacon-API endpoint to use (beacon namespace required)
	BeaconHeader           string   // Optional HTTP header for all requests to L1 Beacon
	BeaconFallbackAddrs    []string // Addresses of L1 Beacon-API fallback endpoints (only for blob sidecars retrieval)
	BeaconCheckIgnore      bool     // When false, halt startup if the beacon version endpoint fails
	BeaconFetchAllSidecars bool     // Whether to fetch all blob sidecars and filter locally
}

var _ L1BeaconEndpointSetup = (*L1BeaconEndpointConfig)(nil)

func (cfg *L1BeaconEndpointConfig) Setup(ctx context.Context, log log.Logger) (cl apis.BeaconClient, fb []apis.BlobSideCarsClient, err error) {
	var opts []client.BasicHTTPClientOption
	if cfg.BeaconHeader != "" {
		hdr, err := parseHTTPHeader(cfg.BeaconHeader)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing beacon header: %w", err)
		}
		opts = append(opts, client.WithHeader(hdr))
	}

	for _, addr := range cfg.BeaconFallbackAddrs {
		b := client.NewBasicHTTPClient(addr, log)
		fb = append(fb, sources.NewBeaconHTTPClient(b))
	}

	a := client.NewBasicHTTPClient(cfg.BeaconAddr, log, opts...)
	return sources.NewBeaconHTTPClient(a), fb, nil
}

func (cfg *L1BeaconEndpointConfig) Check() error {
	if cfg.BeaconAddr == "" && !cfg.BeaconCheckIgnore {
		return errors.New("expected L1 Beacon API endpoint, but got none")
	}
	return nil
}

func (cfg *L1BeaconEndpointConfig) ShouldIgnoreBeaconCheck() bool {
	return cfg.BeaconCheckIgnore
}

func (cfg *L1BeaconEndpointConfig) ShouldFetchAllSidecars() bool {
	return cfg.BeaconFetchAllSidecars
}

func parseHTTPHeader(headerStr string) (http.Header, error) {
	h := make(http.Header, 1)
	s := strings.SplitN(headerStr, ": ", 2)
	if len(s) != 2 {
		return nil, errors.New("invalid header format")
	}
	h.Add(s[0], s[1])
	return h, nil
}
