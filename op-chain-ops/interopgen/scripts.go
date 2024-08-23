package interopgen

type DeployScript struct {
	DeploySafe func(name string) error

	// setupSuperchain
	SetupSuperchain func() error

	// technically still part of setupOpChain, but we deploy them once, and share them, OPSM style.
	// Run this with env "SUPERCHAIN_IMPLEMENTATIONS_WORKAROUND"
	// to not run deployAnchorStateRegistry and deployDelayedWETH.
	DeployImplementations func() error

	// setupOpChain prep, not shared with superchain here, unique per L2
	DeployAddressManager        func() error
	DeployProxyAdmin            func() error
	TransferProxyAdminOwnership func() error

	// setupOpChain core
	DeployProxies             func() error
	DeployDelayedWETH         func() error // work around, address depends on config
	DeployAnchorStateRegistry func() error // work around, depends on a proxy
	InitializeImplementations func() error

	// FP functions
	SetAlphabetFaultGameImplementation           func(allowUpgrade bool) error
	SetFastFaultGameImplementation               func(allowUpgrade bool) error
	SetCannonFaultGameImplementation             func(allowUpgrade bool) error
	SetPermissionedCannonFaultGameImplementation func(allowUpgrade bool) error
	TransferDisputeGameFactoryOwnership          func() error
	TransferDelayedWETHOwnership                 func() error
}

type L2GenesisScript struct {
	RunWithEnv     func() error
	SetPreinstalls func() error
}
