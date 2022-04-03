module github.com/ethereum-optimism/optimism/go/stackman

go 1.16

require (
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/stretchr/testify v1.7.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.7 // required to fix CVE-2021-38561

replace github.com/miekg/dns v1.0.14 => github.com/miekg/dns v1.1.25-0.20191211073109-8ebf2e419df7 // required to fix CVE-2019-19794
