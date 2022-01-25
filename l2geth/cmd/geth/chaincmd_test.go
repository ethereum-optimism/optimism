package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestChainInit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		f, err := os.Open("testdata/init.json")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		io.Copy(w, f)
	}))

	tests := []struct {
		name     string
		url      string
		hash     string
		errorMsg string
	}{
		{
			"no genesis hash specified",
			server.URL,
			"",
			"Must specify a genesis hash argument if the genesis path argument is an URL",
		},
		{
			"invalid genesis hash specified",
			server.URL,
			"not hex yo",
			"Error decoding genesis hash",
		},
		{
			"bad URL",
			"https://honk",
			"0x1234",
			"Failed to fetch genesis file",
		},
		{
			"mis-matched hashes",
			server.URL,
			"0x1234",
			"Genesis hashes do not match",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datadir := tmpdir(t)
			geth := runGeth(t, "init", tt.url, tt.hash, "--datadir", datadir)
			geth.ExpectRegexp(tt.errorMsg)
		})
	}

	t.Run("URL and hash args OK", func(t *testing.T) {
		datadir := tmpdir(t)
		geth := runGeth(t, "init", server.URL, "0x1f0201852c30e203a701ac283aeafafaf55b2ad3ae2f4e8f15c61e761434fb62", "--datadir", datadir)
		geth.ExpectExit()
		geth = runGeth(t, "dump-chain-cfg", "--datadir", datadir)
		geth.ExpectRegexp("\"muirGlacierBlock\": 500")
	})

	t.Run("file arg OK", func(t *testing.T) {
		datadir := tmpdir(t)
		geth := runGeth(t, "init", "testdata/init.json", "--datadir", datadir)
		geth.ExpectExit()
		geth = runGeth(t, "dump-chain-cfg", "--datadir", datadir)
		geth.ExpectRegexp("\"muirGlacierBlock\": 500")
	})
}

func TestDumpChainCfg(t *testing.T) {
	datadir := tmpdir(t)
	geth := runGeth(t, "init", "testdata/init.json", "--datadir", datadir)
	geth.ExpectExit()
	geth = runGeth(t, "dump-chain-cfg", "--datadir", datadir)
	geth.Expect(`{
  "chainId": 69,
  "homesteadBlock": 0,
  "eip150Block": 0,
  "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "eip155Block": 0,
  "eip158Block": 0,
  "byzantiumBlock": 0,
  "constantinopleBlock": 0,
  "petersburgBlock": 0,
  "istanbulBlock": 0,
  "muirGlacierBlock": 500,
  "clique": {
    "period": 0,
    "epoch": 30000
  }
}`)
}
