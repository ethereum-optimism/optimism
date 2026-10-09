import ExporterEvm.Export

/-!
# Concrete runs (reachability witnesses and negative checks)

EVMLean is executable. These runs execute the pinned exporter bytecode with `Ξ` on concrete
states: the exporter code at 0x4200…0030 (run directly, standing in for the proxy's
`DELEGATECALL` frame); a mock L2ToL2CrossDomainMessenger at 0x4200…0023 whose code returns the
word `1` iff the calldata word at offset 4 equals a fixed hash `Hrel`
(`PUSH32 Hrel PUSH1 4 CALLDATALOAD EQ PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN`), i.e.
`successfulMessages(H) = (H == Hrel)`; and a mock L2CrossDomainMessenger at 0x4200…0007 whose code
stores `keccak256(calldata)` in its slot 0 and stops, so the run records exactly what the
messenger received.

Arguments: `_sourceMessenger = 0x1111`, `_source = 901`, `_nonce = 7`, `_sender = 0xaaaa`,
`_target = 0xbbbb`, `_message = 0x0102…25` (37 bytes, so the ABI padding is exercised),
`_minGasLimit = 200000`; `block.timestamp = 1700000000`; EVMLean's chain id is 1.

Reference values computed independently with foundry (`cast`):
* `Hcast = cast keccak $(cast abi-encode "f(uint256,uint256,uint256,address,address,bytes)" 1 901
  7 0x…aaaa 0x…bbbb 0x0102…25)` = `0xe8834c9a…ff2c` (the messenger's message hash with this chain
  as the destination);
* `KSM = cast keccak $(cast calldata "sendMessage(address,bytes,uint32)" 0x…1111 $(cast calldata
  "relayUndeliveredMessage(bytes32,uint256)" Hcast 1700000000) 200000)` = `0x7b18faca…08c7`.

The `*_matches_cast` theorems are kernel-checked (`decide +kernel`); the runs use `native_decide`
(compiled evaluation of `Ξ`) and are tests, not dependencies of the headline theorems.
-/

namespace ExporterEvm.Concrete

open Ethereum Ethereum.EVM Mem

def exporterAddr : AccountAddress :=
  AccountAddress.ofUInt256 (UInt256.ofNat 0x4200000000000000000000000000000000000030)

def Hcast : UInt256 :=
  UInt256.ofNat 0xe8834c9a5006289d9104e9034706b89f5ca7bef422f98a3e21e87379d582ff2c

def KSM : UInt256 :=
  UInt256.ofNat 0x7b18facac444c05bfeddf58c309293965966373919af29cdccb9728328b408c7

def w (n : ℕ) : ByteArray := UInt256.toByteArray (UInt256.ofNat n)

def msgBytes : ByteArray := ⟨((List.range 37).map (fun i => UInt8.ofNat (i + 1))).toArray⟩

/-- `exportUndeliveredMessage(0x1111, 901, 7, sender, 0xbbbb, msgBytes, 200000)` (324 bytes; the
    same bytes as `cast calldata …` for `sender = 0xaaaa`). -/
def calldataOf (sender : ℕ) : ByteArray :=
  ⟨#[0x18, 0x6e, 0x73, 0x28]⟩ ++ w 0x1111 ++ w 901 ++ w 7 ++ w sender ++ w 0xbbbb ++ w 0xe0 ++
    w 200000 ++ w 37 ++ msgBytes ++ ByteArray.zeroes 27

def env (sender : ℕ) (value : ℕ) (perm : Bool) : ExecutionEnv :=
  { (default : ExecutionEnv) with
      codeOwner := exporterAddr
      sender := AccountAddress.ofUInt256 (UInt256.ofNat 0x99)
      source := AccountAddress.ofUInt256 (UInt256.ofNat 0x99)
      weiValue := UInt256.ofNat value
      calldata := calldataOf sender
      code := exporterRuntime
      depth := 1
      perm := perm
      header := { (default : BlockHeader) with timestamp := 1700000000 } }

/-- The witness environment. -/
abbrev env0 : ExecutionEnv := env 0xaaaa 0 true

/-- `PUSH32 Hrel PUSH1 4 CALLDATALOAD EQ PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN`. -/
def mockL2l2Code (Hrel : UInt256) : ByteArray :=
  ⟨#[0x7f]⟩ ++ UInt256.toByteArray Hrel ++
    ⟨#[0x60, 0x04, 0x35, 0x14, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]⟩

/-- `CALLDATASIZE PUSH1 0 PUSH1 0 CALLDATACOPY CALLDATASIZE PUSH1 0 KECCAK256 PUSH1 0 SSTORE STOP`. -/
def mockCdmCode : ByteArray :=
  ⟨#[0x36, 0x60, 0x00, 0x60, 0x00, 0x37, 0x36, 0x60, 0x00, 0x20, 0x60, 0x00, 0x55, 0x00]⟩

/-- `PUSH4 0xdeadbeef PUSH1 0 MSTORE PUSH1 4 PUSH1 28 REVERT`: reverts with `0xdeadbeef`. -/
def revertCdmCode : ByteArray :=
  ⟨#[0x63, 0xde, 0xad, 0xbe, 0xef, 0x60, 0x00, 0x52, 0x60, 0x04, 0x60, 0x1c, 0xfd]⟩

def σx (l2l2Code cdmCode : ByteArray) : AccountMap :=
  (∅ : AccountMap)
    |>.insert exporterAddr { (default : Account) with code := exporterRuntime }
    |>.insert l2l2 { (default : Account) with code := l2l2Code }
    |>.insert l2cdm { (default : Account) with code := cdmCode }

/-- The success world: the mock messenger reports "not relayed" for `Hcast` (it compares with
    `Hcast + 1`). -/
def σ0 : AccountMap := σx (mockL2l2Code (Hcast + UInt256.ofNat 1)) mockCdmCode

def runx (σi : AccountMap) (I : ExecutionEnv) :=
  Ξ σi σi (UInt256.ofNat 10000000) default I

/-- After a successful run: the output is `abi.encode(H)`, the mock messenger's slot 0 holds
    `keccak256` of the calldata it received, and the last log entry is `e`. -/
def successCheck (σi : AccountMap) (I : ExecutionEnv) (H slot0 : UInt256) (e : LogEntry) : Bool :=
  match runx σi I with
  | .ok (.success (σ', _, A') o) =>
      (o == UInt256.toByteArray H) && ((σ'.getD l2cdm default).storage.getD ⟨0⟩ ⟨0⟩ == slot0) &&
        (A'.logSeries.back? == some e)
  | _ => false

def revertOut (σi : AccountMap) (I : ExecutionEnv) : Option ByteArray :=
  match runx σi I with
  | .ok (.revert _ o) => some o
  | _ => none

/-- The statement's preimage definition reproduces Solidity's `abi.encode` (and hence
    `Hashing.hashL2toL2CrossDomainMessage(block.chainid, …)`) on the witness: kernel-checked. -/
theorem exportHash_matches_cast : exportHash env0 = Hcast := by decide +kernel

/-- The statement's `sendMessageCd` layout reproduces Solidity's `abi.encodeCall` for
    `sendMessage(…, abi.encodeCall(relayUndeliveredMessage, (H, t)), …)`: kernel-checked. -/
theorem sendMessageCd_matches_cast :
    UInt256.ofNat (fromByteArrayBigEndian (KEC (sendMessageCd (UInt256.ofNat 0x1111) Hcast
      (UInt256.ofNat 1700000000) (UInt256.ofNat 200000)))) = KSM := by decide +kernel

/-- Reachability witness (success): the output is the 32-byte hash, the mock messenger received
    exactly the `sendMessage` calldata (keccak `KSM`), and the last log entry is the
    `UndeliveredMessageExported` event. -/
theorem success_reachable : successCheck σ0 env0 Hcast KSM (exportedLog env0 Hcast) = true := by
  native_decide

/-- Relayed: the mock reports `successfulMessages(Hcast) = true`; the run reverts with exactly
    `UndeliveredMessageExporter_MessageRelayed()` = `0xccc3f3b0`. -/
theorem relayed_reverts : revertOut (σx (mockL2l2Code Hcast) mockCdmCode) env0 = some messageRelayedError := by
  native_decide

/-- No code at 0x4200…0007: solc's `EXTCODESIZE` check reverts with empty output. -/
theorem no_code_reverts :
    revertOut (σx (mockL2l2Code (Hcast + UInt256.ofNat 1)) ByteArray.empty) env0 = some ByteArray.empty := by
  native_decide

/-- A failing `sendMessage` is bubbled: the revert output is the callee's `0xdeadbeef`. -/
theorem callee_revert_bubbles :
    revertOut (σx (mockL2l2Code (Hcast + UInt256.ofNat 1)) revertCdmCode) env0 =
      some ⟨#[0xde, 0xad, 0xbe, 0xef]⟩ := by
  native_decide

/-- A dirty `_sender` (bit 160 set) is rejected by the ABI decoder (empty revert). -/
theorem dirty_sender_reverts :
    revertOut σ0 (env (2 ^ 160 + 0xaaaa) 0 true) = some ByteArray.empty := by native_decide

/-- ETH attached: the non-payable check reverts (empty). -/
theorem value_reverts : revertOut σ0 (env 0xaaaa 1 true) = some ByteArray.empty := by native_decide

/-- A static frame (`perm = false`) with a messenger whose code just stops: the `LOG3` raises a
    static-mode violation. -/
theorem static_violation :
    (match runx (σx (mockL2l2Code (Hcast + UInt256.ofNat 1)) ⟨#[0x00]⟩) (env 0xaaaa 0 false) with
     | .error .StaticModeViolation => true | _ => false) = true := by native_decide

end ExporterEvm.Concrete
