module github.com/ethereum-optimism/optimism/go/op-exporter

go 1.16

require (
	github.com/ethereum-optimism/optimism/go/op_exporter v0.0.0-20211207210647-c5a8db939ad4
	github.com/ethereum/go-ethereum v1.10.4
	github.com/prometheus/client_golang v1.4.0
	github.com/sirupsen/logrus v1.4.2
	github.com/ybbus/jsonrpc v2.1.2+incompatible
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
)

replace github.com/docker/docker v1.4.2-0.20180625184442-8e610b2b55bf => github.com/docker/docker v1.6.1 // required to fix CVE-2015-3627

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.7 // required to fix CVE-2021-38561

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/ethereum/go-ethereum v1.10.9 // required to fix CVE-2021-39137; CVE-2021-41173

replace github.com/graph-gophers/graphql-go v0.0.0-20201113091052-beb923fada29 => github.com/graph-gophers/graphql-go v1.3.0 // required to fix CVE-2022-21708
