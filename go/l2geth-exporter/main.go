package main

import (
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/optimisticben/optimism/go/l2geth-exporter/l1contracts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = ":9100"
	}

	gethUrl := os.Getenv("GETH_URL")
	if gethUrl == "" {
		log.Error("GETH_URL environmental variable is required")
		os.Exit(1)
	}
	ovmCtcAddress := os.Getenv("OVM_CTC_ADDRESS")
	if ovmCtcAddress == "" {
		log.Error("OVM_CTC_ADDRESS environmental variable is required")
		os.Exit(1)
	}
	client, err := ethclient.Dial(gethUrl)
	if err != nil {
		log.Error("Problem connecting to GETH: %s", err)
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
	go getCTCTotalElements(ovmCtcAddress, client)

	log.Info("Listening on", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Error("Can't start http server: %s", err)
	}

}

func getCTCTotalElements(address string, client *ethclient.Client) {
	ovmCTC := l1contracts.OVMCTC{
		Address: common.HexToAddress(address),
		Client:  client,
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C

		totalElements, err := ovmCTC.GetTotalElements()
		if err != nil {
			log.Error("Error calling GetTotalElements: %s", err)
			continue
		}
		totalElementsFloat, _ := new(big.Float).SetInt(totalElements).Float64()
		ovmctcTotalElements.WithLabelValues(
			"latest").Set(totalElementsFloat)

	}
}
