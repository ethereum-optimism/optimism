module github.com/ethereum-optimism/optimism/go/bss-core

go 1.16

require (
	github.com/decred/dcrd/hdkeychain/v3 v3.0.0
	github.com/ethereum-optimism/optimism/l2geth v1.0.0
	github.com/ethereum/go-ethereum v1.10.12
	github.com/getsentry/sentry-go v0.11.0
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/tyler-smith/go-bip39 v1.0.1-0.20181017060643-dbb3b84ba2ef
)

replace github.com/ethereum-optimism/optimism/l2geth => ../../l2geth
