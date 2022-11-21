package main

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth-exporter/l1contracts"
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
	ctcAddress := os.Getenv("OVM_CTC_ADDRESS")
	if ctcAddress == "" {
		log.Error("OVM_CTC_ADDRESS environmental variable is required")
		os.Exit(1)
	}
	sccAddress := os.Getenv("OVM_SCC_ADDRESS")
	if sccAddress == "" {
		log.Error("OVM_SCC_ADDRESS environmental variable is required")
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
	go getCTCTotalElements(ctcAddress, "ctc", client)
	go getSCCTotalElements(sccAddress, "scc", client)

	log.Info("Program starting", "listenAddress", listenAddress, "GETH_URL", l1Url, "CTC_ADDRESS", ctcAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Error("Can't start http server", "error", err)
	}

}

func getSCCTotalElements(address string, addressLabel string, client *ethclient.Client) {
	scc := l1contracts.SCC{
		Address: common.HexToAddress(address),
		Client:  client,
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(l1TimeoutSeconds))
		totalElements, err := scc.GetTotalElements(ctx)
		if err != nil {
			addressTotalElementsCallStatus.WithLabelValues("error", addressLabel).Inc()
			log.Error("Error calling GetTotalElements", "address", addressLabel, "error", err)
			cancel()
			continue
		}
		addressTotalElementsCallStatus.WithLabelValues("success", addressLabel).Inc()
		totalElementsFloat, _ := new(big.Float).SetInt(totalElements).Float64()
		addressTotalElements.WithLabelValues("latest", addressLabel).Set(totalElementsFloat)

		log.Info(addressLabel, "TotalElements", totalElementsFloat)
		cancel()
		<-ticker.C

	}
}

func getCTCTotalElements(address string, addressLabel string, client *ethclient.Client) {
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
			addressTotalElementsCallStatus.WithLabelValues("error", addressLabel).Inc()
			log.Error("Error calling GetTotalElements", "address", addressLabel, "error", err)
			cancel()
			continue
		}
		addressTotalElementsCallStatus.WithLabelValues("success", addressLabel).Inc()
		totalElementsFloat, _ := new(big.Float).SetInt(totalElements).Float64()
		addressTotalElements.WithLabelValues("latest", addressLabel).Set(totalElementsFloat)

		log.Info(addressLabel, "TotalElements", totalElementsFloat)
		cancel()
		<-ticker.C

	}
}
