package metrics

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// weiToEther divides the wei value by 10^18 to get a number in ether as a float64
func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}

// LaunchBalanceMetrics fires off a go rountine that queries the balance of the supplied account & periodically records it
// to the balance metric of the namespace. The balance of the account is recorded in Ether (not Wei).
// Cancel the supplied context to shut down the go routine
func LaunchBalanceMetrics(ctx context.Context, log log.Logger, r *prometheus.Registry, ns string, client *ethclient.Client, account common.Address) {
	go func() {
		balanceGuage := promauto.With(r).NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "balance",
			Help:      "balance (in ether) of account " + account.String(),
		})

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
				bigBal, err := client.BalanceAt(ctx, account, nil)
				if err != nil {
					log.Warn("failed to get balance of account", "err", err, "address", account)
					cancel()
					continue
				}
				bal := weiToEther(bigBal)
				balanceGuage.Set(bal)
				cancel()
			case <-ctx.Done():
				log.Info("balance metrics shutting down")
				return
			}
		}

	}()
}
