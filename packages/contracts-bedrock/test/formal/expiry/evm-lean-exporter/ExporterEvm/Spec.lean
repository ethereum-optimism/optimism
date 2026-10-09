import Reasoning.Solc
import Reasoning.Storage
import ExporterEvm.Bytecode
import ExporterEvm.Mem

/-!
# Statement vocabulary for `UndeliveredMessageExporter.exportUndeliveredMessage`

Everything a headline theorem mentions is defined here: the calldata fields, the ABI conditions,
the exact message-hash preimage, the two external calls with their exact calldata, the event, and
the possible revert outputs.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-- `bytes4(keccak256("exportUndeliveredMessage(address,uint256,uint256,address,address,bytes,uint32)"))`
    = `0x186e7328`. -/
def exportSelector : UInt256 := UInt256.ofNat 0x186e7328

abbrev selectorWord (I : ExecutionEnv) : UInt256 := solcSelectorWord I

/-- The calldata word at byte offset `off` (`CALLDATALOAD(off)`, zero-padded past the end). -/
def argWord (I : ExecutionEnv) (off : ℕ) : UInt256 := uInt256OfByteArray (I.calldata.readBytes off 32)

/-- `_sourceMessenger` (head word 0). -/
def argSrcMessenger (I : ExecutionEnv) : UInt256 := argWord I 4
/-- `_source` (head word 1). -/
def argSource (I : ExecutionEnv) : UInt256 := argWord I 36
/-- `_nonce` (head word 2). -/
def argNonce (I : ExecutionEnv) : UInt256 := argWord I 68
/-- `_sender` (head word 3). -/
def argSender (I : ExecutionEnv) : UInt256 := argWord I 100
/-- `_target` (head word 4). -/
def argTarget (I : ExecutionEnv) : UInt256 := argWord I 132
/-- The ABI offset of `_message` (head word 5), relative to the start of the arguments (byte 4). -/
def argMsgOffset (I : ExecutionEnv) : UInt256 := argWord I 164
/-- `_minGasLimit` (head word 6). -/
def argMinGas (I : ExecutionEnv) : UInt256 := argWord I 196

/-- Calldata position of `_message`'s length word: `4 + offset`. -/
def msgPos (I : ExecutionEnv) : ℕ := 4 + (argMsgOffset I).toNat
/-- `_message.length` as the word the code reads (`CALLDATALOAD(4 + offset)`). -/
def argMsgLen (I : ExecutionEnv) : UInt256 := argWord I (msgPos I)
/-- `_message.length` as a number. -/
def msgLen (I : ExecutionEnv) : ℕ := (argMsgLen I).toNat
/-- `_message`: the `msgLen I` calldata bytes after the length word. -/
def argMessage (I : ExecutionEnv) : ByteArray :=
  I.calldata.extract (msgPos I + 32) (msgPos I + 32 + msgLen I)

/-- `(n + 31) / 32 * 32`: `n` rounded up to a multiple of 32 (the ABI's padding). -/
def roundUp32 (n : ℕ) : ℕ := (n + 31) / 32 * 32

/-- `type(uint160).max`, solc's address mask. -/
def addrMask : UInt256 := UInt256.ofNat 1461501637330902918203684832716283019655932542975

/-- The ABI-level conditions of a call to `exportUndeliveredMessage`, exactly as the compiled
    decoder checks them (it reverts with empty output if any fails):

    * no ETH attached (the contract has no payable function; the check is at pc 5);
    * at least `4 + 7·32 = 228` bytes of calldata;
    * the three `address` arguments are clean (high 96 bits zero);
    * the `bytes` offset is at most `2^64 - 1`, its length word lies inside the calldata
      (`4 + offset + 31 < calldatasize`), the length is at most `2^64 - 1`, and the bytes lie
      inside the calldata (`4 + offset + 32 + length ≤ calldatasize`);
    * `_minGasLimit` is a clean `uint32`.

    (Under the headline hypothesis `calldatasize < 2^63` solc's signed comparisons coincide with
    these unsigned ones.) -/
structure ArgsOk (I : ExecutionEnv) : Prop where
  noValue : I.weiValue = ⟨0⟩
  calldataLen : 228 ≤ I.calldata.size
  srcMessengerClean : (argSrcMessenger I).toNat < 2 ^ 160
  senderClean : (argSender I).toNat < 2 ^ 160
  targetClean : (argTarget I).toNat < 2 ^ 160
  offsetOk : (argMsgOffset I).toNat ≤ 2 ^ 64 - 1
  lenWordInside : msgPos I + 31 < I.calldata.size
  lenOk : msgLen I ≤ 2 ^ 64 - 1
  bytesInside : msgPos I + 32 + msgLen I ≤ I.calldata.size
  minGasClean : (argMinGas I).toNat < 2 ^ 32

/-- `block.chainid` as EVMLean's `CHAINID` returns it: the constant `Ethereum.chainId` (EVMLean
    fixes the chain id; it is not part of the execution environment). -/
def chainIdWord : UInt256 := UInt256.ofNat Ethereum.chainId

/-- `block.timestamp` (`TIMESTAMP`). -/
def tsWord (I : ExecutionEnv) : UInt256 := UInt256.ofNat I.header.timestamp

/-- **The exact preimage** of the message hash: `abi.encode(block.chainid, _source, _nonce,
    _sender, _target, _message)`, i.e. what `Hashing.hashL2toL2CrossDomainMessage(block.chainid,
    _source, _nonce, _sender, _target, _message)` hashes (`224 + roundUp32(len)` bytes):

    | bytes                 | content                                          |
    |-----------------------|--------------------------------------------------|
    | 0..32                 | `block.chainid` (the destination)                |
    | 32..64                | `_source`                                        |
    | 64..96                | `_nonce`                                         |
    | 96..128               | `_sender`                                        |
    | 128..160              | `_target`                                        |
    | 160..192              | `0xc0` (offset of the `bytes` tail)              |
    | 192..224              | `len = _message.length`                          |
    | 224..224+len          | `_message`                                       |
    | ..224+roundUp32(len)  | zero padding                                     |
-/
def exportPreimage (I : ExecutionEnv) : ByteArray :=
  UInt256.toByteArray chainIdWord ++ UInt256.toByteArray (argSource I) ++
    UInt256.toByteArray (argNonce I) ++ UInt256.toByteArray (argSender I) ++
    UInt256.toByteArray (argTarget I) ++ UInt256.toByteArray (UInt256.ofNat 0xc0) ++
    UInt256.toByteArray (argMsgLen I) ++ argMessage I ++
    ByteArray.zeroes (roundUp32 (msgLen I) - msgLen I)

/-- `H`: the message hash, `keccak256(exportPreimage)` as a word (EVMLean's `KEC`). -/
def exportHash (I : ExecutionEnv) : UInt256 :=
  UInt256.ofNat (fromByteArrayBigEndian (KEC (exportPreimage I)))

/-! ## External calls -/

/-- `Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER` = 0x4200…0023. -/
def l2l2Word : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000023
def l2l2 : AccountAddress := AccountAddress.ofUInt256 l2l2Word

/-- `Predeploys.L2_CROSS_DOMAIN_MESSENGER` = 0x4200…0007. -/
def l2cdmWord : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000007
def l2cdm : AccountAddress := AccountAddress.ofUInt256 l2cdmWord

/-- `successfulMessages(bytes32)` selector `0xb1b1b209` (the public mapping's getter). -/
def successfulSelector : ByteArray := ⟨#[0xb1, 0xb1, 0xb2, 0x09]⟩

/-- Calldata of `successfulMessages(H)`: selector ‖ `H` (36 bytes). -/
def successfulCalldata (H : UInt256) : ByteArray := successfulSelector ++ UInt256.toByteArray H

/-- `relayUndeliveredMessage(bytes32,uint256)` selector `0x372293c3`. -/
def relaySelector : ByteArray := ⟨#[0x37, 0x22, 0x93, 0xc3]⟩

/-- `abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (H, t))` (68 bytes). -/
def relayMessage (H t : UInt256) : ByteArray :=
  relaySelector ++ UInt256.toByteArray H ++ UInt256.toByteArray t

/-- `sendMessage(address,bytes,uint32)` selector `0x3dbb202b`. -/
def sendMessageSelector : ByteArray := ⟨#[0x3d, 0xbb, 0x20, 0x2b]⟩

/-- Calldata of `L2CrossDomainMessenger.sendMessage(target, relayMessage H t, gasLimit)`
    (228 bytes):

    | bytes    | content                                        |
    |----------|------------------------------------------------|
    | 0..4     | `0x3dbb202b`                                   |
    | 4..36    | `_target` (the source chain's L1 messenger)    |
    | 36..68   | `0x60` (offset of the `bytes`)                 |
    | 68..100  | `_minGasLimit`                                 |
    | 100..132 | `0x44` (= 68, length of the `bytes`)           |
    | 132..200 | `relayUndeliveredMessage(H, t)` (68 bytes)     |
    | 200..228 | 28 zero bytes                                  |
-/
def sendMessageCd (target H t gasLimit : UInt256) : ByteArray :=
  sendMessageSelector ++ UInt256.toByteArray target ++ UInt256.toByteArray (UInt256.ofNat 0x60) ++
    UInt256.toByteArray gasLimit ++ UInt256.toByteArray (UInt256.ofNat 0x44) ++ relayMessage H t ++
    ByteArray.zeroes 28

/-- `StaticCall σ₀ I target cd σc σ' z o`: EVMLean's message-call function `Θ`, run from account
    map `σc` for a `STATICCALL` (`w = false`, value 0) from the executing account to `target`
    with calldata `cd`, under *some* substate and *some* call gas, returns account map `σ'`,
    success flag `z` and output `o`. -/
def StaticCall (σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray)
    (σc σ' : AccountMap) (z : Bool) (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in
      (AccountAddress.ofUInt256 (UInt256.ofNat I.codeOwner)) I.sender target
      (toExecute σc target) callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd
      (I.depth + 1) I.header I.blobVersionedHashes I.blocks false

/-- `CallTo σ₀ I target cd σc σ' z o`: EVMLean's `Θ`, run from account map `σc` for a `CALL` with
    value 0 (and the frame's permission `I.perm`) from the executing account to `target` with
    calldata `cd`, under *some* substate and call gas, returns `σ'`, success flag `z`, output `o`. -/
def CallTo (σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray)
    (σc σ' : AccountMap) (z : Bool) (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in
      (AccountAddress.ofUInt256 (UInt256.ofNat I.codeOwner)) I.sender target
      (toExecute σc target) callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd
      (I.depth + 1) I.header I.blobVersionedHashes I.blocks I.perm

/-- The first 32 bytes of a call's return data as a word (what solc's ABI decoder reads). -/
def returnWord (o : ByteArray) : UInt256 := UInt256.ofNat (fromBytesBigEndian (o.data.toList.take 32))

/-! ## Event and revert outputs -/

/-- `keccak256("UndeliveredMessageExported(bytes32,uint256,address,uint256)")`. -/
def exportedTopic : UInt256 :=
  UInt256.ofNat 0x1f8c424adc6dfc9721173830dfcfcc32cfa4495584f4e8eda281baddc12bfc6d

/-- The `UndeliveredMessageExported(H, source, sourceMessenger, t)` log entry the code emits from
    the executing account: topics `[exportedTopic, H, _source]`, data
    `abi.encode(_sourceMessenger, block.timestamp)` (64 bytes). -/
def exportedLog (I : ExecutionEnv) (H : UInt256) : LogEntry :=
  ⟨I.codeOwner, #[exportedTopic, H, argSource I],
    UInt256.toByteArray (argSrcMessenger I) ++ UInt256.toByteArray (tsWord I)⟩

/-- `UndeliveredMessageExporter_MessageRelayed()` selector `0xccc3f3b0`: the revert output when
    the message was relayed (exactly these 4 bytes). -/
def messageRelayedError : ByteArray := ⟨#[0xcc, 0xc3, 0xf3, 0xb0]⟩

/-- What `RETURNDATACOPY(0, 0, n); REVERT(0, n)` with `n = RETURNDATASIZE` outputs for return
    data `rd`: `rd` itself. (EVMLean's memory read panics, i.e. returns the empty array, on a
    length ≥ 2^64; such a return data size is not reachable with real gas limits.) -/
def bubble (rd : ByteArray) : ByteArray := if rd.size < 2 ^ 64 then rd else ByteArray.empty

end ExporterEvm
