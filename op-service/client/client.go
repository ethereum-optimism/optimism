package client

import (
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

func CookiesRPCOption() (rpc.ClientOption, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return rpc.WithHTTPClient(&http.Client{Jar: jar}), nil
}

// ParseHttpHeader takes a slice of strings of the form "K=V" and returns a http.Header
func ParseHttpHeader(slice []string) http.Header {
	if len(slice) == 0 {
		return nil
	}
	header := make(http.Header)
	for _, s := range slice {
		split := strings.SplitN(s, "=", 2)
		val := ""
		if len(split) >= 2 {
			val = split[1]
		}
		header.Add(split[0], val)
	}
	return header
}
