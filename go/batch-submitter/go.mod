module github.com/ethereum-optimism/optimism/go/batch-submitter

go 1.16

require (
	github.com/ethereum-optimism/optimism/go/bss-core v0.0.0
	github.com/ethereum-optimism/optimism/l2geth v1.0.0
	github.com/ethereum/go-ethereum v1.10.16
	github.com/getsentry/sentry-go v0.11.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/microcosm-cc/bluemonday v1.0.16 // indirect
	github.com/nats-io/jwt v1.2.2 // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/ethereum-optimism/optimism/l2geth => ../../l2geth

replace github.com/ethereum-optimism/optimism/go/bss-core => ../bss-core
