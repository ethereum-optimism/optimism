package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/go/op_exporter/k8sClient"
	"github.com/ethereum-optimism/optimism/go/op_exporter/version"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc"
	"gopkg.in/alecthomas/kingpin.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var UnknownStatus = "UNKNOWN"

var (
	listenAddress = kingpin.Flag(
		"web.listen-address",
		"Address on which to expose metrics and web interface.",
	).Default(":9100").String()
	rpcProvider = kingpin.Flag(
		"rpc.provider",
		"Address for RPC provider.",
	).Default("http://127.0.0.1:8545").String()
	networkLabel = kingpin.Flag(
		"label.network",
		"Label to apply to the metrics to identify the network.",
	).Default("mainnet").String()
	versionFlag = kingpin.Flag(
		"version",
		"Display binary version.",
	).Default("False").Bool()
	unhealthyTimePeriod = kingpin.Flag(
		"wait.minutes",
		"Number of minutes to wait for the next block before marking provider unhealthy.",
	).Default("10").Int()
	sequencerPollingSeconds = kingpin.Flag(
		"sequencer.polling",
		"Number of seconds to wait between sequencer polling cycles.",
	).Default("30").Int()
	enableK8sQuery = kingpin.Flag(
		"k8s.enable",
		"Enable kubernetes info lookup.",
	).Default("false").Bool()
)

type healthCheck struct {
	mu             *sync.RWMutex
	height         uint64
	healthy        bool
	updateTime     time.Time
	allowedMethods []string
	version        *string
}

func healthHandler(health *healthCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health.mu.RLock()
		defer health.mu.RUnlock()
		w.Write([]byte(fmt.Sprintf(`{ "healthy": "%t", "version": "%s" }`, health.healthy, *health.version)))
	}
}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	if *versionFlag {
		fmt.Printf("(version=%s, gitcommit=%s)\n", version.Version, version.GitCommit)
		fmt.Printf("(go=%s, user=%s, date=%s)\n", version.GoVersion, version.BuildUser, version.BuildDate)
		os.Exit(0)
	}
	log.Infoln("exporter config", *listenAddress, *rpcProvider, *networkLabel)
	log.Infoln("Starting op_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())
	opExporterVersion.WithLabelValues(
		strings.Trim(version.Version, "\""), version.GitCommit, version.GoVersion, version.BuildDate).Inc()
	health := healthCheck{
		mu:             new(sync.RWMutex),
		height:         0,
		healthy:        false,
		updateTime:     time.Now(),
		allowedMethods: nil,
		version:        &UnknownStatus,
	}
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/health", healthHandler(&health))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>OP Exporter</title></head>
		<body>
		<h1>OP Exporter</h1>
		<p><a href="/metrics">Metrics</a></p>
		<p><a href="/health">Health</a></p>
		</body>
		</html>`))
	})
	go getRollupGasPrices()
	go getBlockNumber(&health)
	if *enableK8sQuery {
		client, err := k8sClient.Newk8sClient()
		if err != nil {
			log.Fatal(err)
		}
		go getSequencerVersion(&health, client)
	}
	log.Infoln("Listening on", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}

}

func getSequencerVersion(health *healthCheck, client *kubernetes.Clientset) {
	ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		log.Fatalf("Unable to read namespace file: %s", err)
	}
	ticker := time.NewTicker(30 * time.Second)
	for {
		<-ticker.C
		getOpts := metav1.GetOptions{
			TypeMeta:        metav1.TypeMeta{},
			ResourceVersion: "",
		}
		sequencerStatefulSet, err := client.AppsV1().StatefulSets(string(ns)).Get(context.TODO(), "sequencer", getOpts)
		if err != nil {
			health.version = &UnknownStatus
			log.Errorf("Unable to retrieve a sequencer StatefulSet: %s", err)
			continue
		}
		for _, c := range sequencerStatefulSet.Spec.Template.Spec.Containers {
			log.Infof("Checking container %s", c.Name)
			switch {
			case c.Name == "sequencer":
				log.Infof("The sequencer version is: %s", c.Image)
				health.version = &c.Image
			default:
				log.Infof("Unable to find the sequencer container in the statefulset?!?")
			}
		}

	}
}

func getBlockNumber(health *healthCheck) {
	rpcClient := jsonrpc.NewClientWithOpts(*rpcProvider, &jsonrpc.RPCClientOpts{})
	var blockNumberResponse *string
	for {
		if err := rpcClient.CallFor(&blockNumberResponse, "eth_blockNumber"); err != nil {
			health.mu.Lock()
			health.healthy = false
			health.mu.Unlock()
			log.Warnln("Error calling eth_blockNumber, setting unhealthy", err)
		} else {
			log.Infoln("Got block number: ", *blockNumberResponse)
			health.mu.Lock()
			currentHeight, err := hexutil.DecodeUint64(*blockNumberResponse)
			blockNumber.WithLabelValues(
				*networkLabel, "layer2").Set(float64(currentHeight))
			if err != nil {
				log.Warnln("Error decoding block height", err)
				continue
			}
			lastHeight := health.height
			// If the currentHeight is the same as the lastHeight, check that
			// the unhealthyTimePeriod has passed and update health.healthy
			if currentHeight == lastHeight {
				currentTime := time.Now()
				lastTime := health.updateTime
				log.Warnln(fmt.Sprintf("Heights are the same, %v, %v", currentTime, lastTime))
				if lastTime.Add(time.Duration(*unhealthyTimePeriod) * time.Minute).Before(currentTime) {
					health.healthy = false
					log.Warnln("Heights are the same for the unhealthyTimePeriod, setting unhealthy")
				}
			} else {
				log.Warnln("New block height detected, setting healthy")
				health.height = currentHeight
				health.updateTime = time.Now()
				health.healthy = true
			}
			if health.healthy {
				healthySequencer.WithLabelValues(
					*networkLabel).Set(1)
			} else {
				healthySequencer.WithLabelValues(
					*networkLabel).Set(0)
			}

			health.mu.Unlock()
		}
		time.Sleep(time.Duration(*sequencerPollingSeconds) * time.Second)
	}
}

func getRollupGasPrices() {
	rpcClient := jsonrpc.NewClientWithOpts(*rpcProvider, &jsonrpc.RPCClientOpts{})
	var rollupGasPrices *GetRollupGasPrices
	for {
		if err := rpcClient.CallFor(&rollupGasPrices, "rollup_gasPrices"); err != nil {
			log.Warnln("Error calling rollup_gasPrices", err)
		} else {
			l1GasPriceString := rollupGasPrices.L1GasPrice
			l1GasPrice, err := hexutil.DecodeUint64(l1GasPriceString)
			if err != nil {
				log.Warnln("Error converting gasPrice " + l1GasPriceString)
			}
			gasPrice.WithLabelValues(
				*networkLabel, "layer1").Set(float64(l1GasPrice))
			l2GasPriceString := rollupGasPrices.L2GasPrice
			l2GasPrice, err := hexutil.DecodeUint64(l2GasPriceString)
			if err != nil {
				log.Warnln("Error converting gasPrice " + l2GasPriceString)
			}
			gasPrice.WithLabelValues(
				*networkLabel, "layer2").Set(float64(l2GasPrice))
			log.Infoln("Got L1 gas string: ", l1GasPriceString)
			log.Infoln("Got L1 gas prices: ", l1GasPrice)
			log.Infoln("Got L2 gas string: ", l2GasPriceString)
			log.Infoln("Got L2 gas prices: ", l2GasPrice)
		}
		time.Sleep(time.Duration(30) * time.Second)
	}
}
