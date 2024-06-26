package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// LaunchBalanceMetrics starts a periodic query of the balance of the supplied account and records it
// to the "balance" metric of the namespace. The balance of the account is recorded in Ether (not Wei).
// Cancel the supplied context to shut down the go routine
func LaunchBalanceMetrics(log log.Logger, r *prometheus.Registry, ns string, client *ethclient.Client, account common.Address) *clock.LoopFn {
	balanceGauge := promauto.With(r).NewGauge(prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "balance",
		Help:      "balance (in ether) of account " + account.String(),
	})
	return clock.NewLoopFn(clock.SystemClock, func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		bigBal, err := client.BalanceAt(ctx, account, nil)
		if err != nil {
			log.Warn("failed to get balance of account", "err", err, "address", account)
			return
		}
		bal := eth.WeiToEther(bigBal)
		balanceGauge.Set(bal)
	}, func() error {
		log.Info("balance metrics shutting down")
		return nil
	}, 10*time.Second)
}
