package main

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/go/l2geth-exporter/l1contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	l1TimeoutSeconds = 5
)

func main() {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = ":9100"
	}

	log.Root().SetHandler(log.CallerFileHandler(log.StdoutHandler))

	l1Url := os.Getenv("L1_URL")
	if l1Url == "" {
		log.Error("L1_URL environmental variable is required")
		os.Exit(1)
	}
	ctcAddress := os.Getenv("CTC_ADDRESS")
	if ctcAddress == "" {
		log.Error("CTC_ADDRESS environmental variable is required")
		os.Exit(1)
	}
	client, err := ethclient.Dial(l1Url)
	if err != nil {
		log.Error("Problem connecting to L1: %s", err)
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>L2geth Exporter</title></head>
		<body>
		<h1>L2geth Exporter</h1>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})
	go getCTCTotalElements(ctcAddress, client)

	log.Info("Program starting", "listenAddress", listenAddress, "GETH_URL", l1Url, "CTC_ADDRESS", ctcAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Error("Can't start http server", "error", err)
	}

}

func getCTCTotalElements(address string, client *ethclient.Client) {
	ctc := l1contracts.CTC{
		Address: common.HexToAddress(address),
		Client:  client,
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(l1TimeoutSeconds))
		totalElements, err := ctc.GetTotalElements(ctx)
		if err != nil {
			ctcTotalElementsCallSuccess.Set(0)
			log.Error("Error calling GetTotalElements", "error", err)
			cancel()
			continue
		}
		ctcTotalElementsCallSuccess.Set(1)
		totalElementsFloat, _ := new(big.Float).SetInt(totalElements).Float64()
		ctcTotalElements.WithLabelValues(
			"latest").Set(totalElementsFloat)
		log.Info("ctc updated", "ctcTotalElements", totalElementsFloat)
		cancel()
		<-ticker.C

	}
}
