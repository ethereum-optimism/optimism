package fetch

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/ethereum-optimism/optimism/op-fetcher/chaininfo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// fakeL1 is an l1State over fake contracts. Like the EVM, a call to an address without a
// contract succeeds with no return data, and a call a contract does not answer reverts.
type fakeL1 struct {
	contracts map[common.Address]*fakeContract
}

type fakeContract struct {
	// returns maps calldata to return data.
	returns map[string][]byte
	// errs maps calldata to an error other than a revert.
	errs    map[string]error
	storage map[common.Hash]common.Hash
}

func newFakeL1() *fakeL1 {
	return &fakeL1{contracts: make(map[common.Address]*fakeContract)}
}

func (l *fakeL1) contract(addr common.Address) *fakeContract {
	c, ok := l.contracts[addr]
	if !ok {
		c = &fakeContract{
			returns: make(map[string][]byte),
			errs:    make(map[string]error),
			storage: make(map[common.Hash]common.Hash),
		}
		l.contracts[addr] = c
	}
	return c
}

func (l *fakeL1) Call(_ context.Context, to common.Address, data []byte) ([]byte, error) {
	c, ok := l.contracts[to]
	if !ok {
		return nil, nil
	}
	if err, ok := c.errs[string(data)]; ok {
		return nil, err
	}
	if ret, ok := c.returns[string(data)]; ok {
		return ret, nil
	}
	return nil, revertError{errors.New("execution reverted")}
}

func (l *fakeL1) StorageAt(_ context.Context, addr common.Address, slot common.Hash) (common.Hash, error) {
	c, ok := l.contracts[addr]
	if !ok {
		return common.Hash{}, nil
	}
	return c.storage[slot], nil
}

func (c *fakeContract) address(sig string, addr common.Address) *fakeContract {
	c.returns[string(selector(sig))] = common.LeftPadBytes(addr.Bytes(), 32)
	return c
}

func (c *fakeContract) raw(sig string, ret []byte) *fakeContract {
	c.returns[string(selector(sig))] = ret
	return c
}

func (c *fakeContract) gameType(sig string, gameType uint32) *fakeContract {
	return c.raw(sig, uint32Word(gameType))
}

func (c *fakeContract) gameImpl(gameType uint32, impl common.Address) *fakeContract {
	c.returns[string(encodeGameTypeCall("gameImpls(uint32)", gameType))] = common.LeftPadBytes(impl.Bytes(), 32)
	return c
}

func (c *fakeContract) gameArgs(gameType uint32, args []byte) *fakeContract {
	ret := make([]byte, 64, 64+len(args)+32)
	ret[31] = 32
	binary.BigEndian.PutUint64(ret[56:], uint64(len(args)))
	ret = append(ret, args...)
	ret = append(ret, make([]byte, (32-len(args)%32)%32)...)
	c.returns[string(encodeGameTypeCall("gameArgs(uint32)", gameType))] = ret
	return c
}

func uint32Word(v uint32) []byte {
	w := make([]byte, 32)
	binary.BigEndian.PutUint32(w[28:], v)
	return w
}

func packGameArgs(vm, anchorStateRegistry, weth common.Address, roles ...common.Address) []byte {
	args := append(crypto.Keccak256([]byte("prestate")), vm.Bytes()...)
	args = append(args, anchorStateRegistry.Bytes()...)
	args = append(args, weth.Bytes()...)
	args = append(args, uint32Word(901)...)
	for _, r := range roles {
		args = append(args, r.Bytes()...)
	}
	return args
}

func addr(b byte) common.Address {
	return common.Address{19: b}
}

var (
	systemConfig            = addr(0x01)
	l1StandardBridge        = addr(0x02)
	messenger               = addr(0x03)
	portal                  = addr(0x04)
	proxyAdmin              = addr(0x05)
	l1ERC721Bridge          = addr(0x06)
	mintableFactory         = addr(0x07)
	addressManager          = addr(0x08)
	superchainConfig        = addr(0x09)
	ethLockbox              = addr(0x0a)
	portalAnchorStateReg    = addr(0x0b)
	l2OutputOracle          = addr(0x0c)
	disputeGameFactory      = addr(0x0d)
	permissionedGame        = addr(0x0e)
	cannonGame              = addr(0x0f)
	cannonKonaGame          = addr(0x10)
	superPermissionedGame   = addr(0x11)
	superCannonKonaGame     = addr(0x12)
	mips                    = addr(0x13)
	preimageOracle          = addr(0x14)
	gameAnchorStateRegistry = addr(0x15)
	wethPermissioned        = addr(0x16)
	wethPermissionless      = addr(0x17)

	systemConfigOwner = addr(0xa1)
	proxyAdminOwner   = addr(0xa2)
	unsafeBlockSigner = addr(0xa3)
	batcher           = addr(0xa4)
	guardian          = addr(0xa5)
	proposer          = addr(0xa6)
	challenger        = addr(0xa7)
)

// chainInfoInput holds the arguments of fetchChainInfo.
type chainInfoInput struct {
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
}

var input = chainInfoInput{
	SystemConfigProxy:     systemConfig,
	L1StandardBridgeProxy: l1StandardBridge,
}

// newChain returns a chain whose contracts all use their current getters, without a
// DisputeGameFactory.
func newChain() *fakeL1 {
	l1 := newFakeL1()
	l1.contract(systemConfig).
		address("owner()", systemConfigOwner).
		address("unsafeBlockSigner()", unsafeBlockSigner).
		raw("batcherHash()", common.LeftPadBytes(batcher.Bytes(), 32)).
		address("admin()", proxyAdmin).
		address("l1ERC721Bridge()", l1ERC721Bridge).
		address("optimismMintableERC20Factory()", mintableFactory).
		address("disputeGameFactory()", common.Address{})
	l1.contract(proxyAdmin).address("owner()", proxyAdminOwner)
	l1.contract(l1StandardBridge).address("messenger()", messenger)
	l1.contract(messenger).address("portal()", portal)
	key := append(common.LeftPadBytes(messenger.Bytes(), 32), common.LeftPadBytes([]byte{1}, 32)...)
	l1.contract(messenger).storage[crypto.Keccak256Hash(key)] = common.BytesToHash(addressManager.Bytes())
	l1.contract(portal).
		address("guardian()", guardian).
		address("ethLockbox()", ethLockbox).
		address("superchainConfig()", superchainConfig)
	return l1
}

// baseOutput is the output for newChain, before any fault proof fields.
func baseOutput() chaininfo.Info {
	var out chaininfo.Info
	out.SystemConfigProxy = systemConfig
	out.SystemConfigOwner = systemConfigOwner
	out.UnsafeBlockSigner = unsafeBlockSigner
	out.BatchSubmitter = batcher
	out.OpChainProxyAdminImpl = proxyAdmin
	out.OpChainProxyAdminOwner = proxyAdminOwner
	out.L1Erc721BridgeProxy = l1ERC721Bridge
	out.OptimismMintableErc20FactoryProxy = mintableFactory
	out.L1StandardBridgeProxy = l1StandardBridge
	out.L1CrossDomainMessengerProxy = messenger
	out.AddressManagerImpl = addressManager
	out.OptimismPortalProxy = portal
	out.OpChainGuardian = guardian
	out.EthLockboxProxy = ethLockbox
	out.SuperchainConfigProxy = superchainConfig
	return out
}

// withFaultProofs registers a DisputeGameFactory and a respected game type.
func withFaultProofs(l1 *fakeL1, respectedGameType uint32) *fakeContract {
	l1.contract(systemConfig).address("disputeGameFactory()", disputeGameFactory)
	l1.contract(portal).gameType("respectedGameType()", respectedGameType)
	l1.contract(mips).address("oracle()", preimageOracle)
	return l1.contract(disputeGameFactory)
}

// withPermissionedGameGetters registers a permissioned game that predates game args.
func withPermissionedGameGetters(l1 *fakeL1) {
	l1.contract(disputeGameFactory).gameImpl(gameTypePermissionedCannon, permissionedGame)
	l1.contract(permissionedGame).
		address("challenger()", challenger).
		address("proposer()", proposer).
		address("weth()", wethPermissioned).
		address("anchorStateRegistry()", gameAnchorStateRegistry).
		address("vm()", mips)
}

func fetch(t *testing.T, l1 l1State) chaininfo.Info {
	t.Helper()
	out, err := fetchChainInfo(context.Background(), l1, input.SystemConfigProxy, input.L1StandardBridgeProxy)
	require.NoError(t, err)
	return out
}

func TestFetchChainInfo_LegacyGetters(t *testing.T) {
	l1 := newFakeL1()
	l1.contract(systemConfig).
		address("owner()", systemConfigOwner).
		address("unsafeBlockSigner()", unsafeBlockSigner).
		raw("batcherHash()", common.LeftPadBytes(batcher.Bytes(), 32)).
		address("admin()", proxyAdmin)
	l1.contract(proxyAdmin).address("owner()", proxyAdminOwner)
	l1.contract(l1StandardBridge).address("messenger()", messenger)
	l1.contract(messenger).address("PORTAL()", portal)
	l1.contract(portal).
		address("GUARDIAN()", guardian).
		address("L2_ORACLE()", l2OutputOracle)
	l1.contract(l2OutputOracle).address("PROPOSER()", proposer)

	want := baseOutput()
	want.L1Erc721BridgeProxy = common.Address{}
	want.OptimismMintableErc20FactoryProxy = common.Address{}
	want.AddressManagerImpl = common.Address{}
	want.EthLockboxProxy = common.Address{}
	want.SuperchainConfigProxy = common.Address{}
	want.L2OutputOracleProxy = l2OutputOracle
	want.Proposer = proposer
	want.RespectedGameType = math.MaxUint32
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_L2OutputOracle(t *testing.T) {
	l1 := newChain()
	l1.contract(portal).address("l2Oracle()", l2OutputOracle)
	l1.contract(l2OutputOracle).address("PROPOSER()", proposer)

	want := baseOutput()
	want.L2OutputOracleProxy = l2OutputOracle
	want.Proposer = proposer
	want.RespectedGameType = math.MaxUint32
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_RespectedGameTypeWithoutFactory(t *testing.T) {
	l1 := newChain()
	l1.contract(portal).
		gameType("respectedGameType()", gameTypeCannon).
		address("L2_ORACLE()", l2OutputOracle)
	l1.contract(l2OutputOracle).address("PROPOSER()", proposer)

	want := baseOutput()
	want.L2OutputOracleProxy = l2OutputOracle
	want.Proposer = proposer
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_PermissionedGetters(t *testing.T) {
	l1 := newChain()
	withFaultProofs(l1, gameTypePermissionedCannon)
	withPermissionedGameGetters(l1)

	want := baseOutput()
	want.RespectedGameType = gameTypePermissionedCannon
	want.DisputeGameFactoryProxy = disputeGameFactory
	want.Permissioned = true
	want.PermissionedDisputeGameImpl = permissionedGame
	want.Challenger = challenger
	want.Proposer = proposer
	want.DelayedWethPermissionedGameProxy = wethPermissioned
	want.AnchorStateRegistryProxy = gameAnchorStateRegistry
	want.MipsImpl = mips
	want.PreimageOracleImpl = preimageOracle
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_PermissionlessGetters(t *testing.T) {
	tests := []struct {
		name               string
		cannon, cannonKona common.Address
		wantFaultGame      common.Address
	}{
		{name: "cannon", cannon: cannonGame, wantFaultGame: cannonGame},
		{name: "cannon and cannon kona", cannon: cannonGame, cannonKona: cannonKonaGame, wantFaultGame: cannonGame},
		{name: "cannon kona only", cannonKona: cannonKonaGame, wantFaultGame: cannonKonaGame},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l1 := newChain()
			factory := withFaultProofs(l1, gameTypeCannon)
			withPermissionedGameGetters(l1)
			l1.contract(portal).address("anchorStateRegistry()", portalAnchorStateReg)
			for gameType, impl := range map[uint32]common.Address{gameTypeCannon: tt.cannon, gameTypeCannonKona: tt.cannonKona} {
				if impl != (common.Address{}) {
					factory.gameImpl(gameType, impl)
					l1.contract(impl).address("weth()", wethPermissionless)
				}
			}

			want := baseOutput()
			want.RespectedGameType = gameTypeCannon
			want.DisputeGameFactoryProxy = disputeGameFactory
			want.Permissioned = true
			want.PermissionedDisputeGameImpl = permissionedGame
			want.Challenger = challenger
			want.Proposer = proposer
			want.DelayedWethPermissionedGameProxy = wethPermissioned
			want.Permissionless = true
			want.FaultDisputeGameImpl = tt.wantFaultGame
			want.FaultDisputeGameCannonKonaImpl = tt.cannonKona
			want.DelayedWethPermissionlessGameProxy = wethPermissionless
			want.AnchorStateRegistryProxy = portalAnchorStateReg
			want.MipsImpl = mips
			want.PreimageOracleImpl = preimageOracle
			require.Equal(t, want, fetch(t, l1))
		})
	}
}

func TestFetchChainInfo_PermissionlessOnlyGetters(t *testing.T) {
	l1 := newChain()
	factory := withFaultProofs(l1, gameTypeCannon)
	factory.gameImpl(gameTypeCannon, cannonGame)
	// A weth() that does not return exactly one word reads as no DelayedWETH.
	l1.contract(cannonGame).
		raw("weth()", make([]byte, 64)).
		address("vm()", mips)

	want := baseOutput()
	want.RespectedGameType = gameTypeCannon
	want.DisputeGameFactoryProxy = disputeGameFactory
	want.Permissionless = true
	want.FaultDisputeGameImpl = cannonGame
	want.MipsImpl = mips
	want.PreimageOracleImpl = preimageOracle
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_GameArgs(t *testing.T) {
	l1 := newChain()
	factory := withFaultProofs(l1, gameTypeCannonKona)
	factory.
		gameImpl(gameTypePermissionedCannon, permissionedGame).
		gameArgs(gameTypePermissionedCannon, packGameArgs(mips, gameAnchorStateRegistry, wethPermissioned, proposer, challenger)).
		gameImpl(gameTypeCannonKona, cannonKonaGame).
		gameArgs(gameTypeCannonKona, packGameArgs(addr(0xee), addr(0xef), wethPermissionless))

	want := baseOutput()
	want.RespectedGameType = gameTypeCannonKona
	want.DisputeGameFactoryProxy = disputeGameFactory
	want.Permissioned = true
	want.PermissionedDisputeGameImpl = permissionedGame
	want.Challenger = challenger
	want.Proposer = proposer
	want.DelayedWethPermissionedGameProxy = wethPermissioned
	want.Permissionless = true
	want.FaultDisputeGameImpl = cannonKonaGame
	want.FaultDisputeGameCannonKonaImpl = cannonKonaGame
	want.DelayedWethPermissionlessGameProxy = wethPermissionless
	// The permissioned game's args name the AnchorStateRegistry and VM first.
	want.AnchorStateRegistryProxy = gameAnchorStateRegistry
	want.MipsImpl = mips
	want.PreimageOracleImpl = preimageOracle
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_SuperRootGames(t *testing.T) {
	l1 := newChain()
	factory := withFaultProofs(l1, gameTypeSuperPermissioned)
	factory.
		gameImpl(gameTypeSuperPermissioned, superPermissionedGame).
		gameArgs(gameTypeSuperPermissioned, append(gameAnchorStateRegistry.Bytes(), proposer.Bytes()...)).
		gameImpl(gameTypeSuperCannonKona, superCannonKonaGame).
		gameArgs(gameTypeSuperCannonKona, packGameArgs(mips, addr(0xef), wethPermissionless))

	want := baseOutput()
	want.RespectedGameType = gameTypeSuperPermissioned
	want.DisputeGameFactoryProxy = disputeGameFactory
	want.Permissioned = true
	want.SuperPermissionedDisputeGameImpl = superPermissionedGame
	want.Proposer = proposer
	want.Permissionless = true
	want.SuperFaultDisputeGameImpl = superCannonKonaGame
	want.DelayedWethPermissionlessGameProxy = wethPermissionless
	want.AnchorStateRegistryProxy = gameAnchorStateRegistry
	want.MipsImpl = mips
	want.PreimageOracleImpl = preimageOracle
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_PartialGameConfig(t *testing.T) {
	l1 := newChain()
	factory := withFaultProofs(l1, gameTypePermissionedCannon)
	factory.
		gameImpl(gameTypePermissionedCannon, permissionedGame).
		// Permissionless-length args name no proposer or challenger.
		gameArgs(gameTypePermissionedCannon, packGameArgs(mips, gameAnchorStateRegistry, wethPermissioned)).
		gameImpl(gameTypeCannon, cannonGame).
		// A super game without args contributes only its implementation.
		gameImpl(gameTypeSuperCannonKona, superCannonKonaGame)
	// A weth() that reverts reads as no DelayedWETH.
	l1.contract(cannonGame).address("vm()", mips)

	want := baseOutput()
	want.RespectedGameType = gameTypePermissionedCannon
	want.DisputeGameFactoryProxy = disputeGameFactory
	want.Permissioned = true
	want.PermissionedDisputeGameImpl = permissionedGame
	want.DelayedWethPermissionedGameProxy = wethPermissioned
	want.Permissionless = true
	want.FaultDisputeGameImpl = cannonGame
	want.SuperFaultDisputeGameImpl = superCannonKonaGame
	want.AnchorStateRegistryProxy = gameAnchorStateRegistry
	want.MipsImpl = mips
	want.PreimageOracleImpl = preimageOracle
	require.Equal(t, want, fetch(t, l1))
}

func TestFetchChainInfo_PortalAnchorStateRegistryWins(t *testing.T) {
	l1 := newChain()
	l1.contract(portal).address("anchorStateRegistry()", portalAnchorStateReg)
	factory := withFaultProofs(l1, gameTypeSuperPermissioned)
	factory.
		gameImpl(gameTypeSuperPermissioned, superPermissionedGame).
		gameArgs(gameTypeSuperPermissioned, append(gameAnchorStateRegistry.Bytes(), proposer.Bytes()...))

	out := fetch(t, l1)
	require.Equal(t, portalAnchorStateReg, out.AnchorStateRegistryProxy)
}

func TestFetchChainInfo_Errors(t *testing.T) {
	errTransport := errors.New("connection reset")
	tests := []struct {
		name    string
		setup   func(l1 *fakeL1)
		input   chainInfoInput
		wantErr error
	}{
		{
			name:  "zero SystemConfig",
			input: chainInfoInput{L1StandardBridgeProxy: l1StandardBridge},
		},
		{
			name:  "zero L1StandardBridge",
			input: chainInfoInput{SystemConfigProxy: systemConfig},
		},
		{
			name: "required getter reverts",
			setup: func(l1 *fakeL1) {
				delete(l1.contract(systemConfig).returns, string(selector("unsafeBlockSigner()")))
			},
		},
		{
			name: "legacy fallback reverts",
			setup: func(l1 *fakeL1) {
				delete(l1.contract(portal).returns, string(selector("guardian()")))
			},
		},
		{
			name: "call to an address without code",
			setup: func(l1 *fakeL1) {
				l1.contract(l1StandardBridge).address("messenger()", addr(0xff))
			},
		},
		{
			name: "transport error is not a revert",
			setup: func(l1 *fakeL1) {
				l1.contract(systemConfig).errs[string(selector("l1ERC721Bridge()"))] = errTransport
			},
			wantErr: errTransport,
		},
		{
			// Misreading this as a revert would report a chain as having no fault proofs.
			name: "transport error on the respected game type",
			setup: func(l1 *fakeL1) {
				l1.contract(portal).errs[string(selector("respectedGameType()"))] = errTransport
			},
			wantErr: errTransport,
		},
		{
			name: "address with dirty upper bytes",
			setup: func(l1 *fakeL1) {
				l1.contract(systemConfig).raw("owner()", common.LeftPadBytes([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 32))
			},
		},
		{
			name: "respected game type out of range",
			setup: func(l1 *fakeL1) {
				l1.contract(portal).raw("respectedGameType()", common.LeftPadBytes([]byte{1, 0, 0, 0, 0}, 32))
			},
		},
		{
			name: "game args of invalid length",
			setup: func(l1 *fakeL1) {
				withFaultProofs(l1, gameTypePermissionedCannon).
					gameImpl(gameTypePermissionedCannon, permissionedGame).
					gameArgs(gameTypePermissionedCannon, make([]byte, 100))
			},
		},
		{
			name: "super permissioned game args of invalid length",
			setup: func(l1 *fakeL1) {
				withFaultProofs(l1, gameTypeSuperPermissioned).
					gameImpl(gameTypeSuperPermissioned, superPermissionedGame).
					gameArgs(gameTypeSuperPermissioned, make([]byte, 41))
			},
		},
		{
			name: "game args with an out-of-bounds length",
			setup: func(l1 *fakeL1) {
				factory := withFaultProofs(l1, gameTypePermissionedCannon).
					gameImpl(gameTypePermissionedCannon, permissionedGame)
				ret := make([]byte, 64)
				ret[31] = 32
				ret[63] = 33
				factory.returns[string(encodeGameTypeCall("gameArgs(uint32)", gameTypePermissionedCannon))] = ret
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l1 := newChain()
			l1.contract(portal).
				gameType("respectedGameType()", gameTypeCannon).
				address("L2_ORACLE()", l2OutputOracle)
			l1.contract(l2OutputOracle).address("PROPOSER()", proposer)
			if tt.setup != nil {
				tt.setup(l1)
			}
			in := input
			if tt.input != (chainInfoInput{}) {
				in = tt.input
			}
			_, err := fetchChainInfo(context.Background(), l1, in.SystemConfigProxy, in.L1StandardBridgeProxy)
			require.Error(t, err)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
