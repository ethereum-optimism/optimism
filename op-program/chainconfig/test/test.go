package test

import "embed"

//go:embed configs/*json
var TestCustomChainConfigFS embed.FS

//go:embed configs_empty/*json
var TestCustomChainConfigEmptyFS embed.FS

//go:embed configs_typo/*json
var TestCustomChainConfigTypoFS embed.FS

//go:embed configs_no_l1/*.json
var TestCustomChainConfigNoL1FS embed.FS

//go:embed configs_multiple_l1/*.json
var TestCustomChainConfigMultipleL1FS embed.FS
