// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"time"
)

type Config struct {
	RPCURL string

	// Timeouts
	DialTimeout      time.Duration
	SubscribeTimeout time.Duration
}

// DefaultConfig returns the default config. Default values can be overwritten with env variables, most importantly
// RPC_URL for a custom RPC endpoint.
func Default() Config {
	return Config{
		RPCURL:           extractDefaultRPCURL(),
		DialTimeout:      10 * time.Second,
		SubscribeTimeout: 5 * time.Second,
	}
}

// ExtractDefaultRPCURL reads the env variable RPC_URL and returns it. If that variable is unset or empty,
// it will fallback to "ws://127.0.0.1:9944"
func extractDefaultRPCURL() string {
	if url, ok := os.LookupEnv("RPC_URL"); ok {
		return url
	}

	// Fallback
	return "ws://127.0.0.1:9944"
}
