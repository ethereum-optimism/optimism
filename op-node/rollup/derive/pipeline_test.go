package derive

import "github.com/HashKeyChain/verse/op-service/testutils"

var _ L1Fetcher = (*testutils.MockL1Source)(nil)

var _ Metrics = (*testutils.TestDerivationMetrics)(nil)
