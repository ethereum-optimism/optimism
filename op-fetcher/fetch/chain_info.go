package fetch

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-fetcher/chaininfo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Dispute game types, as defined in the contracts' GameTypes library.
const (
	gameTypeCannon             uint32 = 0
	gameTypePermissionedCannon uint32 = 1
	gameTypeSuperPermissioned  uint32 = 5
	gameTypeCannonKona         uint32 = 8
	gameTypeSuperCannonKona    uint32 = 9
)

// Lengths of the packed game args accepted by the contracts' LibGameArgs decoders.
const (
	permissionlessArgsLength    = 124
	permissionedArgsLength      = 164
	superPermissionedArgsLength = 40
)

// addressManagerMappingSlot is the storage slot of the ResolvedDelegateProxy's addressManager
// mapping, which the legacy L1CrossDomainMessenger proxy is keyed into.
const addressManagerMappingSlot = 1

// chainInfoReader reads a chain's L1 contracts, roles and fault proof status into out. It
// walks the contracts from the SystemConfig and L1StandardBridge proxies, and supports every
// contract release a live chain may run by falling back to legacy getters where a call
// reverts. The fill methods read the addresses that earlier steps filled into out, and fill
// in what they read from L1.
//
// The first error is sticky: once a read fails, further reads return zero values and the
// result is discarded, which keeps the walk itself free of error plumbing.
type chainInfoReader struct {
	ctx context.Context
	l1  l1State
	out chaininfo.Info
	err error
}

func fetchChainInfo(ctx context.Context, l1 l1State, systemConfig, l1StandardBridge common.Address) (chaininfo.Info, error) {
	if systemConfig == (common.Address{}) {
		return chaininfo.Info{}, errors.New("SystemConfigProxy not set")
	}
	if l1StandardBridge == (common.Address{}) {
		return chaininfo.Info{}, errors.New("L1StandardBridgeProxy not set")
	}
	r := &chainInfoReader{ctx: ctx, l1: l1}
	r.out.SystemConfigProxy = systemConfig
	r.out.L1StandardBridgeProxy = l1StandardBridge
	r.fillSystemConfig()
	r.fillMessengerAndPortal()
	r.fillFaultProofs()
	if r.err != nil {
		return chaininfo.Info{}, r.err
	}
	return r.out, nil
}

func (r *chainInfoReader) fillSystemConfig() {
	out := &r.out
	systemConfig := out.SystemConfigProxy
	out.SystemConfigOwner = r.address(systemConfig, "owner()")
	out.UnsafeBlockSigner = r.address(systemConfig, "unsafeBlockSigner()")
	// The batcher hash is the batcher address, left-padded to 32 bytes.
	out.BatchSubmitter = common.BytesToAddress(r.bytes32(systemConfig, "batcherHash()").Bytes())
	// A proxy answers admin() itself only when called by its admin or the zero address,
	// which is why every call is made from the zero address.
	out.OpChainProxyAdminImpl = r.address(systemConfig, "admin()")
	out.OpChainProxyAdminOwner = r.address(out.OpChainProxyAdminImpl, "owner()")
	out.L1Erc721BridgeProxy = r.tryAddress(systemConfig, "l1ERC721Bridge()")
	out.OptimismMintableErc20FactoryProxy = r.tryAddress(systemConfig, "optimismMintableERC20Factory()")
}

func (r *chainInfoReader) fillMessengerAndPortal() {
	out := &r.out
	messenger := r.address(out.L1StandardBridgeProxy, "messenger()")
	out.L1CrossDomainMessengerProxy = messenger
	out.AddressManagerImpl = r.addressManager(messenger)
	portal := r.addressOr(messenger, "portal()", "PORTAL()")
	out.OptimismPortalProxy = portal
	out.OpChainGuardian = r.addressOr(portal, "guardian()", "GUARDIAN()")
	out.EthLockboxProxy = r.tryAddress(portal, "ethLockbox()")
	// Newer portals point to the AnchorStateRegistry; otherwise a dispute game names it.
	out.AnchorStateRegistryProxy = r.tryAddress(portal, "anchorStateRegistry()")
	out.SuperchainConfigProxy = r.tryAddress(portal, "superchainConfig()")
}

// fillFaultProofs fills the respected game type and either the L2OutputOracle or the
// dispute games. The AnchorStateRegistry and the VM are shared across game types: the first
// read that names one wins, starting with the portal's anchorStateRegistry().
func (r *chainInfoReader) fillFaultProofs() {
	out := &r.out
	portal := out.OptimismPortalProxy

	respectedGameType, ok := r.tryGameType(portal, "respectedGameType()")
	if !ok {
		// A portal that predates fault proofs reads outputs from an L2OutputOracle. The
		// respected game type is uint32.max rather than 0, which is CANNON.
		out.RespectedGameType = math.MaxUint32
		r.fillL2OutputOracle(r.addressOr(portal, "l2Oracle()", "L2_ORACLE()"))
		return
	}
	out.RespectedGameType = respectedGameType

	out.DisputeGameFactoryProxy = r.tryAddress(out.SystemConfigProxy, "disputeGameFactory()")
	if out.DisputeGameFactoryProxy == (common.Address{}) {
		r.fillL2OutputOracle(r.address(portal, "L2_ORACLE()"))
		return
	}

	r.fillPermissionedGame()
	r.fillSuperPermissionedGame()
	r.fillPermissionlessGame()
	r.fillSuperCannonKonaGame()

	if out.MipsImpl != (common.Address{}) {
		out.PreimageOracleImpl = r.address(out.MipsImpl, "oracle()")
	}
}

func (r *chainInfoReader) fillL2OutputOracle(l2OutputOracle common.Address) {
	r.out.L2OutputOracleProxy = l2OutputOracle
	r.out.Proposer = r.address(l2OutputOracle, "PROPOSER()")
}

func (r *chainInfoReader) fillPermissionedGame() {
	out := &r.out
	factory := out.DisputeGameFactoryProxy
	game := r.gameImpl(factory, gameTypePermissionedCannon)
	if game == (common.Address{}) {
		return
	}
	out.Permissioned = true
	out.PermissionedDisputeGameImpl = game
	if args, ok := r.gameArgs(factory, gameTypePermissionedCannon); ok {
		out.Challenger = args.challenger
		out.Proposer = args.proposer
		out.DelayedWethPermissionedGameProxy = args.weth
		setIfZero(&out.AnchorStateRegistryProxy, args.anchorStateRegistry)
		out.MipsImpl = args.vm
		return
	}
	// Games that predate game args expose their configuration as getters.
	out.Challenger = r.address(game, "challenger()")
	out.Proposer = r.address(game, "proposer()")
	out.DelayedWethPermissionedGameProxy = r.delayedWETH(game)
	if out.AnchorStateRegistryProxy == (common.Address{}) {
		out.AnchorStateRegistryProxy = r.address(game, "anchorStateRegistry()")
	}
	out.MipsImpl = r.address(game, "vm()")
}

func (r *chainInfoReader) fillSuperPermissionedGame() {
	out := &r.out
	factory := out.DisputeGameFactoryProxy
	game := r.gameImpl(factory, gameTypeSuperPermissioned)
	if game == (common.Address{}) {
		return
	}
	out.Permissioned = true
	out.SuperPermissionedDisputeGameImpl = game

	raw := r.rawGameArgs(factory, gameTypeSuperPermissioned)
	if len(raw) == 0 {
		return
	}
	if len(raw) != superPermissionedArgsLength {
		r.fail(fmt.Errorf("SUPER_PERMISSIONED game args of factory %s: invalid length %d", factory, len(raw)))
		return
	}
	out.Proposer = common.BytesToAddress(raw[20:40])
	setIfZero(&out.AnchorStateRegistryProxy, common.BytesToAddress(raw[0:20]))
}

// fillPermissionlessGame fills the live permissionless game. CANNON is being replaced by
// CANNON_KONA, so it reports whichever is registered, preferring CANNON when both are, which
// keeps FaultDisputeGameImpl stable for chains that have not migrated.
func (r *chainInfoReader) fillPermissionlessGame() {
	out := &r.out
	factory := out.DisputeGameFactoryProxy
	cannon := r.gameImpl(factory, gameTypeCannon)
	cannonKona := r.gameImpl(factory, gameTypeCannonKona)
	out.FaultDisputeGameCannonKonaImpl = cannonKona
	gameType, game := gameTypeCannon, cannon
	if cannon == (common.Address{}) {
		gameType, game = gameTypeCannonKona, cannonKona
	}
	if game == (common.Address{}) {
		return
	}
	out.FaultDisputeGameImpl = game
	out.Permissionless = true
	if args, ok := r.gameArgs(factory, gameType); ok {
		out.DelayedWethPermissionlessGameProxy = args.weth
		setIfZero(&out.MipsImpl, args.vm)
		setIfZero(&out.AnchorStateRegistryProxy, args.anchorStateRegistry)
		return
	}
	out.DelayedWethPermissionlessGameProxy = r.delayedWETH(game)
	if out.MipsImpl == (common.Address{}) {
		out.MipsImpl = r.address(game, "vm()")
	}
}

func (r *chainInfoReader) fillSuperCannonKonaGame() {
	out := &r.out
	factory := out.DisputeGameFactoryProxy
	game := r.gameImpl(factory, gameTypeSuperCannonKona)
	if game == (common.Address{}) {
		return
	}
	out.SuperFaultDisputeGameImpl = game
	out.Permissionless = true

	args, ok := r.gameArgs(factory, gameTypeSuperCannonKona)
	if !ok {
		return
	}
	out.DelayedWethPermissionlessGameProxy = args.weth
	setIfZero(&out.AnchorStateRegistryProxy, args.anchorStateRegistry)
	setIfZero(&out.MipsImpl, args.vm)
}

// setIfZero sets *addr to v unless an earlier read has already set it. v is evaluated before
// the call, so a getter that must only be called when *addr is zero needs an explicit if.
func setIfZero(addr *common.Address, v common.Address) {
	if *addr == (common.Address{}) {
		*addr = v
	}
}

// gameArgs holds the fields of LibGameArgs.GameArgs that are read here.
type gameArgs struct {
	vm                  common.Address
	anchorStateRegistry common.Address
	weth                common.Address
	proposer            common.Address
	challenger          common.Address
}

// gameArgs returns the decoded game args of a game type, and false if the factory has none
// for it.
func (r *chainInfoReader) gameArgs(factory common.Address, gameType uint32) (gameArgs, bool) {
	raw := r.rawGameArgs(factory, gameType)
	if len(raw) == 0 {
		return gameArgs{}, false
	}
	// The packed layout is absolutePrestate (32), vm, anchorStateRegistry, weth (20 each),
	// l2ChainId (32) and, for permissioned games only, proposer and challenger (20 each).
	if len(raw) != permissionlessArgsLength && len(raw) != permissionedArgsLength {
		r.fail(fmt.Errorf("game args of game type %d of factory %s: invalid length %d", gameType, factory, len(raw)))
		return gameArgs{}, false
	}
	args := gameArgs{
		vm:                  common.BytesToAddress(raw[32:52]),
		anchorStateRegistry: common.BytesToAddress(raw[52:72]),
		weth:                common.BytesToAddress(raw[72:92]),
	}
	if len(raw) == permissionedArgsLength {
		args.proposer = common.BytesToAddress(raw[124:144])
		args.challenger = common.BytesToAddress(raw[144:164])
	}
	return args, true
}

// rawGameArgs returns the packed game args of a game type, or nil if the factory has none
// for it or predates game args.
func (r *chainInfoReader) rawGameArgs(factory common.Address, gameType uint32) []byte {
	ret, err := r.tryCall(factory, "gameArgs(uint32)", encodeGameTypeCall("gameArgs(uint32)", gameType))
	if err != nil {
		return nil
	}
	return r.decodeBytes(factory, "gameArgs(uint32)", ret)
}

// gameImpl returns the implementation of a game type, or the zero address if it is not
// registered or the factory cannot be queried.
func (r *chainInfoReader) gameImpl(factory common.Address, gameType uint32) common.Address {
	ret, err := r.tryCall(factory, "gameImpls(uint32)", encodeGameTypeCall("gameImpls(uint32)", gameType))
	if err != nil {
		return common.Address{}
	}
	return r.decodeAddress(factory, "gameImpls(uint32)", ret)
}

// delayedWETH returns a game's DelayedWETH, or the zero address if the game does not
// return exactly one word for weth().
func (r *chainInfoReader) delayedWETH(game common.Address) common.Address {
	ret, err := r.tryCall(game, "weth()", selector("weth()"))
	if err != nil || len(ret) != 32 {
		return common.Address{}
	}
	return r.decodeAddress(game, "weth()", ret)
}

// addressManager reads the legacy AddressManager from the L1CrossDomainMessenger proxy's
// addressManager mapping, keyed by the proxy's own address.
func (r *chainInfoReader) addressManager(messenger common.Address) common.Address {
	if r.err != nil {
		return common.Address{}
	}
	key := make([]byte, 64)
	copy(key[12:32], messenger.Bytes())
	key[63] = addressManagerMappingSlot
	val, err := r.l1.StorageAt(r.ctx, messenger, crypto.Keccak256Hash(key))
	if err != nil {
		r.fail(fmt.Errorf("failed to read AddressManager of %s: %w", messenger, err))
		return common.Address{}
	}
	return common.BytesToAddress(val.Bytes())
}

// address calls a getter that must succeed.
func (r *chainInfoReader) address(to common.Address, sig string) common.Address {
	ret, err := r.tryCall(to, sig, selector(sig))
	if err != nil {
		r.fail(fmt.Errorf("%s.%s: %w", to, sig, err))
		return common.Address{}
	}
	return r.decodeAddress(to, sig, ret)
}

// tryAddress calls a getter that may revert, returning the zero address if it does.
func (r *chainInfoReader) tryAddress(to common.Address, sig string) common.Address {
	ret, err := r.tryCall(to, sig, selector(sig))
	if err != nil {
		return common.Address{}
	}
	return r.decodeAddress(to, sig, ret)
}

// addressOr calls a getter, falling back to its legacy name if the getter reverts.
func (r *chainInfoReader) addressOr(to common.Address, sig, legacySig string) common.Address {
	ret, err := r.tryCall(to, sig, selector(sig))
	if err != nil {
		return r.address(to, legacySig)
	}
	return r.decodeAddress(to, sig, ret)
}

// tryGameType calls a GameType getter, returning false if it reverts.
func (r *chainInfoReader) tryGameType(to common.Address, sig string) (uint32, bool) {
	ret, err := r.tryCall(to, sig, selector(sig))
	if err != nil {
		return 0, false
	}
	word := r.word(to, sig, ret)
	if !isZero(word[:28]) {
		r.fail(fmt.Errorf("%s.%s: returned value does not fit a uint32: %x", to, sig, word))
		return 0, false
	}
	return binary.BigEndian.Uint32(word[28:]), true
}

func (r *chainInfoReader) bytes32(to common.Address, sig string) common.Hash {
	ret, err := r.tryCall(to, sig, selector(sig))
	if err != nil {
		r.fail(fmt.Errorf("%s.%s: %w", to, sig, err))
		return common.Hash{}
	}
	return common.BytesToHash(r.word(to, sig, ret))
}

// tryCall calls the getter sig of a contract with data. It returns an error if the call
// reverts or the reader has already failed. A failure other than a revert is also recorded
// as the reader's error, so a caller only has to tell a required getter from one that may
// revert.
func (r *chainInfoReader) tryCall(to common.Address, sig string, data []byte) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	ret, err := r.l1.Call(r.ctx, to, data)
	if err != nil && !errors.Is(err, errExecutionReverted) {
		r.fail(fmt.Errorf("failed to call %s.%s: %w", to, sig, err))
	}
	return ret, err
}

// The decoders below accept exactly what the Solidity ABI decoder accepts for a single
// return value: return data that is too short, or an address with dirty upper bytes, is an
// error rather than a revert, because it is not a call failure a fallback could handle.

func (r *chainInfoReader) word(to common.Address, sig string, ret []byte) []byte {
	if r.err != nil {
		return make([]byte, 32)
	}
	if len(ret) < 32 {
		r.fail(fmt.Errorf("%s.%s: return data too short (no contract at the address?): %x", to, sig, ret))
		return make([]byte, 32)
	}
	return ret[:32]
}

func (r *chainInfoReader) decodeAddress(to common.Address, sig string, ret []byte) common.Address {
	word := r.word(to, sig, ret)
	if !isZero(word[:12]) {
		r.fail(fmt.Errorf("%s.%s: returned value is not an address: %x", to, sig, word))
		return common.Address{}
	}
	return common.BytesToAddress(word[12:])
}

func (r *chainInfoReader) decodeBytes(to common.Address, sig string, ret []byte) []byte {
	word := r.word(to, sig, ret)
	if r.err != nil {
		return nil
	}
	offset, ok := wordUint64(word)
	if !ok || offset > uint64(len(ret))-32 {
		r.fail(fmt.Errorf("%s.%s: invalid bytes offset: %x", to, sig, ret))
		return nil
	}
	start := offset + 32
	length, ok := wordUint64(ret[start-32 : start])
	if !ok || length > uint64(len(ret))-start {
		r.fail(fmt.Errorf("%s.%s: invalid bytes length: %x", to, sig, ret))
		return nil
	}
	return ret[start : start+length]
}

// wordUint64 returns a 32-byte word as a uint64, and false if it does not fit.
func wordUint64(word []byte) (uint64, bool) {
	return binary.BigEndian.Uint64(word[24:]), isZero(word[:24])
}

func (r *chainInfoReader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

func selector(sig string) []byte {
	return crypto.Keccak256([]byte(sig))[:4]
}

func encodeGameTypeCall(sig string, gameType uint32) []byte {
	data := make([]byte, 4+32)
	copy(data, selector(sig))
	binary.BigEndian.PutUint32(data[4+28:], gameType)
	return data
}

func isZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}
