module github.com/ethereum-optimism/optimism/go/batch-submitter

go 1.16

require (
	github.com/ethereum-optimism/optimism/go/bss-core v0.0.0
	github.com/ethereum-optimism/optimism/l2geth v1.0.0
	github.com/ethereum/go-ethereum v1.10.16
	github.com/getsentry/sentry-go v0.11.0
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
)

replace github.com/ethereum-optimism/optimism/l2geth => ../../l2geth

replace github.com/ethereum-optimism/optimism/go/bss-core => ../bss-core
