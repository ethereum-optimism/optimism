package dsl

import (
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
)

// EventCertificates drives the trusted-signer prototype through the real L1 settlement path.
// It deliberately submits source calls as deposits, so an offline private sequencer is sufficient.
type EventCertificates struct {
	commonImpl
	source, destination       *L2Network
	projection, destinationEL *L2ELNode
	l1                        *L1ELNode
	submitter                 *EOA
	depositor                 *DepositEOA
	registry, verifier        common.Address
}

// NewEventCertificates deploys the registry and attestation verifier, and configures the inboxes.
// The source must use the projection EL; ordinary private EL receipts cannot identify its outbox.
func NewEventCertificates(t devtest.T, source, destination *L2Network, projection, destinationEL *L2ELNode,
	l1 *L1ELNode, submitter *EOA) *EventCertificates {
	c := &EventCertificates{commonImpl: commonFromT(t), source: source, destination: destination,
		projection: projection, destinationEL: destinationEL, l1: l1, submitter: submitter}
	c.depositor = submitter.AsEL(projection).ViaDepositTx(submitter, projection, source)
	var lockbox common.Address
	c.read(l1, source.DepositContractAddr(), "ethLockbox()", "address", &lockbox)
	if lockbox == (common.Address{}) {
		lockbox = c.configureTestCluster()
	}
	receipt := c.transact(submitter, nil, c.initCode("L1EventRegistry", lockbox))
	c.registry = receipt.ContractAddress
	receipt = c.depositor.DeployContract(c.initCode("AttestedEventVerifier", submitter.Address(), predeploys.CrossL2InboxAddr), 1_500_000)
	c.verifier = receipt.ContractAddress
	c.configure(source, projection, "setL1EventRegistry(address)", c.registry)
	c.configure(source, projection, "setEventProofVerifier(address)", c.verifier)
	c.configure(destination, destinationEL, "setL1EventRegistry(address)", c.registry)
	return c
}

// configureTestCluster equips the isolated devnet's ordinary portals with a real shared lockbox.
// This is fixture initialization through the devnet ProxyAdmin, not a production cluster migration.
// StorageSetter restores the portal implementation in the SAME call, never leaving an open setter.
func (c *EventCertificates) configureTestCluster() common.Address {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	c.require.NoError(err)
	key, err := keys.Secret(devkeys.L1ProxyAdminOwnerRole.Key(c.l1.ChainID().ToBig()))
	c.require.NoError(err)
	admin := NewKey(c.t, key).User(c.l1)
	c.submitter.Transfer(admin.Address(), eth.OneTenthEther)
	portals := []common.Address{c.source.DepositContractAddr(), c.destination.DepositContractAddr()}
	var sourceConfig, sourceAdmin common.Address
	c.read(c.l1, portals[0], "systemConfig()", "address", &sourceConfig)
	c.read(c.l1, portals[0], "proxyAdmin()", "address", &sourceAdmin)
	lockboxImpl := c.transact(c.submitter, nil, c.initCode("ETHLockbox")).ContractAddress
	lockbox := c.transact(c.submitter, nil, c.initCode("Proxy", sourceAdmin)).ContractAddress
	c.transact(admin, &sourceAdmin, c.encode("upgradeAndCall(address,address,bytes)", lockbox, lockboxImpl,
		c.encode("initialize(address,address[])", sourceConfig, portals)))
	setter := c.transact(c.submitter, nil, c.initCode("StorageSetter")).ContractAddress
	layout := c.artifact("OptimismPortal2").StorageLayout
	entry, err := layout.GetStorageLayoutEntry("ethLockbox")
	c.require.NoError(err)
	c.require.Zero(entry.Offset, "fixture expects the lockbox address at offset zero")
	implementationSlot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	var feature common.Hash
	copy(feature[:], "ETH_LOCKBOX")
	type slot struct{ Key, Value common.Hash }
	for _, portal := range portals {
		var config, proxyAdmin, owner, implementation common.Address
		c.read(c.l1, portal, "systemConfig()", "address", &config)
		c.read(c.l1, portal, "proxyAdmin()", "address", &proxyAdmin)
		c.read(c.l1, proxyAdmin, "owner()", "address", &owner)
		c.require.Equal(admin.Address(), owner, "fixture requires the devnet governance key")
		c.read(c.l1, proxyAdmin, "getProxyImplementation(address)", "address", &implementation, portal)
		c.transact(admin, &config, c.encode("setFeature(bytes32,bool)", feature, true))
		lockboxSlot := common.BigToHash(new(big.Int).SetUint64(uint64(entry.Slot)))
		lockboxWord, err := c.l1.EthClient().GetStorageAt(c.ctx, portal, lockboxSlot, "latest")
		c.require.NoError(err)
		copy(lockboxWord[12:], lockbox[:])
		writes := []slot{
			{lockboxSlot, lockboxWord},
			{implementationSlot, common.BytesToHash(implementation.Bytes())},
		}
		c.transact(admin, &proxyAdmin, c.encode("upgradeAndCall(address,address,bytes)", portal, setter,
			c.encode("setBytes32((bytes32 key,bytes32 value)[])", writes)))
		var actual common.Address
		c.read(c.l1, portal, "ethLockbox()", "address", &actual)
		c.require.Equal(lockbox, actual)
		c.read(c.l1, proxyAdmin, "getProxyImplementation(address)", "address", &actual, portal)
		c.require.Equal(implementation, actual, "fixture must restore the canonical portal implementation")
	}
	return lockbox
}

func (c *EventCertificates) configure(network *L2Network, el *L2ELNode, signature string, value common.Address) {
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	c.require.NoError(err)
	key, err := keys.Secret(devkeys.L2ProxyAdminOwnerRole.Key(network.ChainID().ToBig()))
	c.require.NoError(err)
	admin := NewKey(c.t, key).User(c.l1)
	var owner common.Address
	c.read(el, predeploys.ProxyAdminAddr, "owner()", "address", &owner)
	c.require.Equal(owner, admin.Address(), "devnet proxy admin key must match deployment")
	c.submitter.Transfer(admin.Address(), eth.OneHundredthEther)
	admin.AsEL(el).ViaDepositTx(admin, el, network).DepositTx(predeploys.CrossL2InboxAddr, c.encode(signature, value))
}

// AttestedMessage retains the original projected position, not the later export block's position.
type AttestedMessage struct {
	id                 certificateIdentifier
	payload            []byte
	hash               common.Hash
	proof              []byte
	relayHash          common.Hash
	privateOutboxNonce *big.Int
}

type certificateIdentifier struct {
	Origin                                    common.Address
	BlockNumber, LogIndex, Timestamp, ChainId *big.Int
}

// AttestMessage signs an actual private receipt after resolving its public projection position.
// This signer is trusted to check canonicality; the signature itself is not an execution proof.
func (c *EventCertificates) AttestMessage(private *L2ELNode, receipt *types.Receipt) *AttestedMessage {
	c.require.Len(receipt.Logs, 1, "expected a single SentMessage")
	ref := private.BlockRefByNumber(bigs.Uint64Strict(receipt.BlockNumber))
	var output txintent.InteropOutput
	c.require.NoError(output.FromReceipt(c.ctx, receipt, ref.BlockRef(), c.source.ChainID()))
	id := output.Entries[0].Identifier
	m := &AttestedMessage{
		id: certificateIdentifier{id.Origin, new(big.Int).SetUint64(id.BlockNumber), new(big.Int).SetUint64(uint64(id.LogIndex)),
			new(big.Int).SetUint64(id.Timestamp), id.ChainID.ToBig()},
		payload: messages.LogToMessagePayload(receipt.Logs[0]), hash: output.Entries[0].PayloadHash,
	}
	sent, err := render.DecodeSentMessage(receipt.Logs[0].Topics, receipt.Logs[0].Data)
	c.require.NoError(err)
	m.relayHash = crypto.Keccak256Hash(c.encode("message(uint256,uint256,uint256,address,address,bytes)",
		sent.Destination, id.ChainID.ToBig(), sent.Nonce, sent.Sender, sent.Target, sent.Message)[4:])
	c.read(private, predeploys.L2ToL1MessagePasserAddr, "messageNonce()", "uint256", &m.privateOutboxNonce)
	var digest common.Hash
	c.read(c.projection, c.verifier, "eventDigest((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId),bytes32)", "bytes32", &digest, m.id, m.hash)
	signature, err := crypto.Sign(digest[:], c.submitter.Key().Priv())
	c.require.NoError(err)
	signature[64] += 27
	m.proof = signature
	return m
}

// Export force-includes a proof, then reconstructs its projection withdrawal despite suppressed logs.
func (c *EventCertificates) Export(m *AttestedMessage) *Withdrawal {
	var nonce, messengerNonce *big.Int
	var otherMessenger common.Address
	c.read(c.projection, predeploys.L2ToL1MessagePasserAddr, "messageNonce()", "uint256", &nonce)
	c.read(c.projection, predeploys.L2CrossDomainMessengerAddr, "messageNonce()", "uint256", &messengerNonce)
	c.read(c.projection, predeploys.L2CrossDomainMessengerAddr, "otherMessenger()", "address", &otherMessenger)
	inner := c.encode("registerEvent((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId),bytes32)", m.id, m.hash)
	var gas uint64
	c.read(c.projection, predeploys.L2CrossDomainMessengerAddr, "baseGas(bytes,uint32)", "uint64", &gas, inner, uint32(200_000))
	data := c.encode("relayMessage(uint256,address,address,uint256,uint256,bytes)", messengerNonce,
		predeploys.CrossL2InboxAddr, c.registry, new(big.Int), big.NewInt(200_000), inner)
	receipt := c.depositor.DepositTxWithGas(predeploys.ProjectionEventExporterAddr,
		c.encode("exportProvenEvent((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId),bytes32,bytes)", m.id, m.hash, m.proof), 1_000_000)
	c.require.Empty(receipt.Logs, "projection deposits must still suppress receipt logs")
	projectionNetwork := NewL2Network(c.source.Escape(), c.projection, nil, c.l1, nil, nil)
	bridge := NewStandardBridge(c.t, projectionNetwork, c.l1)
	return bridge.TrackWithdrawal(receipt, bindings.WithdrawalTransaction{Nonce: nonce,
		Sender: predeploys.L2CrossDomainMessengerAddr, Target: otherMessenger, Value: new(big.Int),
		GasLimit: new(big.Int).SetUint64(gas), Data: data})
}

// Relay verifies L1 registration and delivers the certificate through the destination portal.
func (c *EventCertificates) Relay(m *AttestedMessage) {
	var certificate common.Hash
	c.read(c.l1, c.registry, "calculateCertificate((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId),bytes32)", "bytes32", &certificate, m.id, m.hash)
	var registered bool
	c.read(c.l1, c.registry, "registeredEvents(bytes32)", "bool", &registered, certificate)
	c.require.True(registered, "finalized projection withdrawal must register the event on L1")
	receipt := c.transact(c.submitter, &c.registry, c.encode("relayMessage(address,(address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId),bytes,uint64)",
		c.destination.DepositContractAddr(), m.id, m.payload, uint64(1_000_000)))
	var hash common.Hash
	for _, log := range receipt.Logs {
		deposit, err := derive.UnmarshalDepositLogEvent(log)
		if err == nil {
			hash = deposit.Hash()
			break
		}
	}
	c.require.NotEqual(common.Hash{}, hash, "registry must deposit the certificate on the destination")
	c.destinationEL.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(receipt.BlockNumber), 120)
	delivered := c.destinationEL.WaitForReceipt(hash)
	c.require.Equal(types.ReceiptStatusSuccessful, delivered.Status)
	certifiedTopic := crypto.Keccak256Hash([]byte("ExecutingCertifiedMessage(bytes32,(address,uint256,uint256,uint256,uint256))"))
	ordinaryTopic := crypto.Keccak256Hash([]byte("ExecutingMessage(bytes32,(address,uint256,uint256,uint256,uint256))"))
	found := false
	for _, log := range delivered.Logs {
		if log.Address != predeploys.CrossL2InboxAddr || len(log.Topics) == 0 {
			continue
		}
		c.require.NotEqual(ordinaryTopic, log.Topics[0], "certificate delivery must not create an ordinary interop dependency")
		if log.Topics[0] == certifiedTopic {
			c.require.Len(log.Topics, 2)
			c.require.Equal(m.hash, log.Topics[1])
			encodedID := c.encode("id((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId))", m.id)[4:]
			c.require.Equal(encodedID, log.Data, "certificate must retain the original event position")
			found = true
		}
	}
	c.require.True(found, "destination must execute the authenticated message")
	var successful bool
	c.read(c.destinationEL, predeploys.L2toL2CrossDomainMessengerAddr, "successfulMessages(bytes32)", "bool", &successful, m.relayHash)
	c.require.True(successful, "destination must record successful execution of this exact message")
}

// VerifyPrivateDepositReverted checks that recovery consumes the deposit without exporting privately.
func (c *EventCertificates) VerifyPrivateDepositReverted(private *L2ELNode, withdrawal *Withdrawal, m *AttestedMessage) {
	projected := c.projection.BlockRefByNumber(bigs.Uint64Strict(withdrawal.initReceipt.BlockNumber))
	private.WaitL1OriginReached(eth.Unsafe, projected.L1Origin.Number, 120)
	receipt := private.WaitForReceipt(withdrawal.InitiateTxHash())
	c.require.Equal(types.ReceiptStatusFailed, receipt.Status, "private projection-export placeholder must revert")
	c.require.Empty(receipt.Logs)
	var nonce *big.Int
	c.read(private, predeploys.L2ToL1MessagePasserAddr, "messageNonce()", "uint256", &nonce)
	c.require.Equal(m.privateOutboxNonce, nonce, "private recovery must not create a withdrawal")
	var consumed bool
	eventID := crypto.Keccak256Hash(c.encode("id((address origin,uint256 blockNumber,uint256 logIndex,uint256 timestamp,uint256 chainId))", m.id)[4:])
	c.read(private, predeploys.CrossL2InboxAddr, "provenEvents(bytes32)", "bool", &consumed, eventID)
	c.require.False(consumed, "private recovery must not consume the certificate")
}

func (c *EventCertificates) encode(signature string, args ...any) []byte {
	data, err := w3.MustNewFunc(signature, "").EncodeArgs(args...)
	c.require.NoError(err)
	return data
}

func (c *EventCertificates) read(el ELNode, address common.Address, signature, result string, out any, args ...any) {
	f := w3.MustNewFunc(signature, result)
	data, err := f.EncodeArgs(args...)
	c.require.NoError(err)
	value, err := el.stackEL().EthClient().Call(c.ctx, ethereum.CallMsg{To: &address, Data: data}, rpc.LatestBlockNumber)
	c.require.NoError(err, "read %s at %s", signature, address)
	c.require.NoError(f.DecodeReturns(value, out))
}

func (c *EventCertificates) transact(user *EOA, to *common.Address, data []byte) *types.Receipt {
	tx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(to), txplan.WithData(data))
	receipt, err := tx.Included.Eval(c.ctx)
	c.require.NoError(err)
	c.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	return receipt
}

func (c *EventCertificates) artifact(name string) *foundry.Artifact {
	wd, err := os.Getwd()
	c.require.NoError(err)
	root, err := opservice.FindMonorepoRoot(wd)
	c.require.NoError(err)
	artifact, err := foundry.ReadArtifact(filepath.Join(root, "packages/contracts-bedrock/forge-artifacts", name+".sol", name+".json"))
	c.require.NoError(err)
	return artifact
}

func (c *EventCertificates) initCode(name string, args ...any) []byte {
	artifact := c.artifact(name)
	constructor, err := artifact.ABI.Pack("", args...)
	c.require.NoError(err)
	bytecode := append([]byte(nil), artifact.Bytecode.Object...)
	c.require.NotEmpty(bytecode)
	return append(bytecode, constructor...)
}
