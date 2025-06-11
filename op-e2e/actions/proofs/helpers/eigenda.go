package helpers

import (
	"os"
)

var eigendaProxyUrl string

func init() {
	eigendaProxyUrl = os.Getenv("EIGENDA_PROXY_URL")
}

func IsEigenDAConfigured() bool {
	return eigendaProxyUrl != ""
}
