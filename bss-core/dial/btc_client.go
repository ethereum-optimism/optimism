package dial

import (
	"github.com/btcsuite/btcd/rpcclient"
)

func BTCClientWithTimeout(url string, postModeHTTP bool) (
	*rpcclient.Client, error) {


	connCfg := &rpcclient.ConnConfig{
		Host:         url,
		User:         "test",
		Pass:         "test",
		HTTPPostMode: postModeHTTP, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   false,
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.

	return rpcclient.New(connCfg, nil)
}