module github.com/ethereum-optimism/optimism/go/bss-core

go 1.16

require (
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/decred/dcrd/hdkeychain/v3 v3.0.0
	github.com/ethereum/go-ethereum v1.10.16
	github.com/getsentry/sentry-go v0.11.0
	github.com/prometheus/client_golang v1.11.0
	github.com/stretchr/testify v1.7.0
	github.com/tyler-smith/go-bip39 v1.0.1-0.20181017060643-dbb3b84ba2ef
)

replace github.com/docker/docker v1.4.2-0.20180625184442-8e610b2b55bf => github.com/docker/docker v1.6.1 // required to fix CVE-2015-3627

replace github.com/gin-gonic/gin v1.4.0 => github.com/gin-gonic/gin v1.6.3-0.20210406033725-bfc8ca285eb4 // indirect; required to fix CVE-2020-28483

replace github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2 // required to fix CVE-2021-3121

replace github.com/microcosm-cc/bluemonday v1.0.2 => github.com/microcosm-cc/bluemonday v1.0.16 // required to fix CVE-2021-42576

replace github.com/nats-io/jwt v0.3.0 => github.com/nats-io/jwt v1.1.0 // required to fix CVE-2020-26892, CVE-2020-26521

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.7 // required to fix CVE-2021-38561
