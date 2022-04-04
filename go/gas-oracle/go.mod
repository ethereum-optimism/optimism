module github.com/ethereum-optimism/optimism/go/gas-oracle

go 1.16

require (
	github.com/ethereum/go-ethereum v1.10.16
	github.com/urfave/cli v1.20.0
)

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.7 // required to fix CVE-2021-38561

replace github.com/docker/docker v1.4.2-0.20180625184442-8e610b2b55bf => github.com/docker/docker v1.6.1 // required to fix CVE-2015-3627
