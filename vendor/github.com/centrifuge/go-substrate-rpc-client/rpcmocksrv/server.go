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

package rpcmocksrv

import (
	"math/rand"
	"strconv"
	"time"

	gethrpc "github.com/centrifuge/go-substrate-rpc-client/gethrpc"
)

type Server struct {
	*gethrpc.Server
	// Host consists of hostname and port
	Host string
	// URL consists of protocol, hostname and port
	URL string
}

// New creates a new RPC mock server with a random port that allows registration of services
func New() *Server {
	port := randomPort()
	host := "localhost:" + strconv.Itoa(port)

	_, rpcServ, err := gethrpc.StartWSEndpoint(host, []gethrpc.API{}, []string{}, []string{"*"}, true)
	if err != nil {
		panic(err)
	}
	s := Server{
		Server: rpcServ,
		Host:   host,
		URL:    "ws://" + host,
	}
	return &s
}

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	min := 10000
	max := 30000
	return rand.Intn(max-min+1) + min
}
