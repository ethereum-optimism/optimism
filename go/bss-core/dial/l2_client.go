package dial

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/l2geth/log"
	"github.com/ethereum-optimism/optimism/l2geth/rpc"
)

// L2EthClientWithTimeout attempts to dial the L2 provider using the
// provided URL. If the dial doesn't complete within defaultDialTimeout seconds,
// this method will return an error.
func L2EthClientWithTimeout(ctx context.Context, url string, disableHTTP2 bool) (
	*ethclient.Client, error) {

	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	if strings.HasPrefix(url, "http") {
		httpClient := new(http.Client)
		if disableHTTP2 {
			log.Info("Disabled HTTP/2 support in L2 eth client")
			httpClient.Transport = &http.Transport{
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			}
		}

		rpcClient, err := rpc.DialHTTPWithClient(url, httpClient)
		if err != nil {
			return nil, err
		}

		return ethclient.NewClient(rpcClient), nil
	}

	return ethclient.DialContext(ctxt, url)
}
