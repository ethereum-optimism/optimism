import Reasoning.Solc
import Reasoning.Storage
import ExpiryEvm.Bytecode

/-!
# Statement vocabulary for `L2ToL2CrossDomainMessenger.expireMessage`

Everything a reader needs to check the theorem statements in `ExpireMessage.lean`, in terms of
the EVMLean semantics (`Ethereum.EVM.Ξ`, `Ethereum.EVM.Θ`, `AccountMap`, `ExecutionEnv`):

* the calldata view (`selectorWord`, `argHash`, `argTime`),
* the contract's storage view (`sentAtSlot`, `expiredSlot`, `selfStorageWord`),
* the external static calls the code makes to the L2CrossDomainMessenger (`L2cdmStaticCall`),
  and the summary hypothesis on their results (`ReturnsAddress`),
* the success conditions (`ExpireConds`) and the post-state relation (`ExpirePost`).

Solidity source (at commit 448d31ad19, tip of `karl/message-expiry-refunds`):

```solidity
function expireMessage(bytes32 _messageHash, uint256 _undeliveredAt) external {
    if (msg.sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER
        || ICrossDomainMessenger(L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()
            != address(ICrossDomainMessenger(L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
    ) revert L2ToL2CrossDomainMessenger_NotOtherMessenger();
    uint256 sentAt = sentMessageTimestamps[_messageHash];
    if (sentAt == 0) revert InvalidMessage();
    if (_undeliveredAt <= sentAt + EXPIRY_PERIOD) revert L2ToL2CrossDomainMessenger_MessageNotExpired();
    expiredMessages[_messageHash] = true;
    emit MessageExpired(_messageHash, _undeliveredAt);
}
```
-/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory

/-! ## Constants of the compiled artifact -/

/-- `EXPIRY_PERIOD` as compiled into the artifact (`PUSH3 0x0a8c00` at pc 2179):
    691200 s = 8 days at commit 448d31ad19 (unchanged since 5992028e08). The proofs refer to this
    name only; if the constant changes, regenerate the bytecode and change this one definition
    (see HOWTO.md). -/
def P_contract : ℕ := 691200

/-- `Predeploys.L2_CROSS_DOMAIN_MESSENGER` = 0x4200000000000000000000000000000000000007, as a word. -/
def l2cdmWord : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000007

/-- The L2CrossDomainMessenger predeploy address. -/
def l2cdm : AccountAddress := AccountAddress.ofUInt256 l2cdmWord

/-- `bytes4(keccak256("expireMessage(bytes32,uint256)"))`. -/
def expireSelector : UInt256 := UInt256.ofNat 0x763a1cb7

/-- Calldata of `otherMessenger()`: its selector `0xdb505d80`. -/
def otherMessengerCalldata : ByteArray := ⟨#[0xdb, 0x50, 0x5d, 0x80]⟩

/-- Calldata of `xDomainMessageSender()`: its selector `0x6e296e45`. -/
def xDomainMessageSenderCalldata : ByteArray := ⟨#[0x6e, 0x29, 0x6e, 0x45]⟩

/-! ## Calldata view -/

/-- The selector word solc dispatches on: `CALLDATALOAD(0) >> 224`. -/
abbrev selectorWord (I : ExecutionEnv) : UInt256 := solcSelectorWord I

/-- First argument, `bytes32 _messageHash` = `CALLDATALOAD(4)`. -/
def argHash (I : ExecutionEnv) : UInt256 := uInt256OfByteArray (I.calldata.readBytes 4 32)

/-- Second argument, `uint256 _undeliveredAt` = `CALLDATALOAD(36)`. -/
def argTime (I : ExecutionEnv) : UInt256 := uInt256OfByteArray (I.calldata.readBytes 36 32)

/-! ## Storage view

Storage layout (`forge inspect L2ToL2CrossDomainMessenger storageLayout`): slot 3 is
`sentMessageTimestamps` (`mapping(bytes32 => uint256)`), slot 4 is `expiredMessages`
(`mapping(bytes32 => bool)`). Solidity puts `m[k]` at `keccak256(k ‖ slot)`. -/

/-- Storage slot of `sentMessageTimestamps[H]`. -/
def sentAtSlot (H : UInt256) : UInt256 := solcMappingSlot (UInt256.ofNat 3) H

/-- Storage slot of `expiredMessages[H]`. -/
def expiredSlot (H : UInt256) : UInt256 := solcMappingSlot (UInt256.ofNat 4) H

/-- Word at `slot` of account `a`'s persistent storage in `σ` (0 if unset or no account). -/
def storageWord (σ : AccountMap) (a : AccountAddress) (slot : UInt256) : UInt256 :=
  (σ.getD a default).storage.getD slot ⟨0⟩

/-- `sentMessageTimestamps[H]` of the executing contract (`I.codeOwner`) in `σ`. -/
def sentAt (σ : AccountMap) (I : ExecutionEnv) : UInt256 :=
  storageWord σ I.codeOwner (sentAtSlot (argHash I))

/-- Solidity's packed-bool write of `true` into a slot whose old word is `old`:
    `(old & ~0xff) | 1`. The compiled code does exactly this (`SLOAD; AND ~0xff; OR 1; SSTORE`),
    so the other 31 bytes of the slot are kept; for a slot only ever written by Solidity as a
    `bool`, `old ∈ {0, 1}` and the new word is `1`. -/
def setTrueWord (old : UInt256) : UInt256 :=
  UInt256.lor (UInt256.ofNat 1)
    -- 2^256 - 256 = 0xffff…ff00 = ~0xff, the PUSH32 mask at pc 2711
    (UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680) old)

/-! ## External calls

`expireMessage` makes two `STATICCALL`s to the L2CrossDomainMessenger at 0x4200..0007, in this
order (legacy solc evaluates the right operand of `!=` first): `otherMessenger()` and then
`xDomainMessageSender()`, each with 4 bytes of calldata and a 32-byte output buffer.

`L2cdmStaticCall σ₀ I cd σc σ' z o` says: the EVMLean message-call function `Θ`, run from account
map `σc` for a static call (`w = false`) from the executing contract to 0x..07 with calldata `cd`
and value 0, under *some* substate and *some* call gas, returns account map `σ'`, success flag `z`
and output `o`. The call gas and substate are existentially quantified, so every statement about
call results holds whatever gas the 63/64 rule forwards. -/
def L2cdmStaticCall (σ₀ : AccountMap) (I : ExecutionEnv) (cd : ByteArray)
    (σc σ' : AccountMap) (z : Bool) (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in
      (AccountAddress.ofUInt256 (UInt256.ofNat I.codeOwner)) I.sender l2cdm
      (toExecute σc l2cdm) callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd
      (I.depth + 1) I.header I.blobVersionedHashes I.blocks false

/-- 32-byte ABI encoding of an address (the address in the low 20 bytes). -/
def abiAddress (v : AccountAddress) : ByteArray := UInt256.toByteArray (UInt256.ofNat v.val)

/-- **Summary (hypothesis) of an L2CrossDomainMessenger view function.** Run from any account map
    with the same storage, transient storage and code as `σ`, a *successful* static call with
    calldata `cd` returns at least 32 bytes whose first 32-byte word is the ABI encoding of `v`
    (any further bytes are unconstrained; the compiled decoder ignores them). Failure
    (`z = false`) is unconstrained.

    This is what the deployed L2CrossDomainMessenger does for `otherMessenger()` and
    `xDomainMessageSender()` (Solidity `address`-returning view functions; the latter reverts
    outside a relay). It is an assumption, not proved here (the L2CrossDomainMessenger bytecode is
    not verified in this development). `v` is thereby the address the code decodes from the
    callee's return data. -/
def ReturnsAddress (σ σ₀ : AccountMap) (I : ExecutionEnv) (cd : ByteArray)
    (v : AccountAddress) : Prop :=
  ∀ σc σ' z o, accountStorageStateEq σ σc → accountCodeStateEq σ σc →
    L2cdmStaticCall σ₀ I cd σc σ' z o → z = true → 32 ≤ o.size ∧ o.extract 0 32 = abiAddress v

/-- A call the code makes to the L2CrossDomainMessenger *can* fail: the call depth limit is
    reached; or the `otherMessenger()` call from `σ` returns `z = false`; or it succeeds (into
    `σ₁`) and the following `xDomainMessageSender()` call from `σ₁` returns `z = false`.

    **Weak**: the forwarded call gas and the substate are existentially quantified (the `RD`
    framework does not expose the gas the 63/64 rule forwards), so this holds in essentially
    every state (e.g. a call with 0 gas fails; `Concrete.callFailed_in_success_state` proves it in
    a state where the run succeeds). Statements with a `revert ∧ CallFailed` disjunct are
    therefore *not* completeness/liveness statements. See README. -/
def CallFailed (σ σ₀ : AccountMap) (I : ExecutionEnv) : Prop :=
  I.depth.val = 1024 ∨
  (∃ σ' o, L2cdmStaticCall σ₀ I otherMessengerCalldata σ σ' false o) ∨
  (∃ σ₁ o₁ σ' o, L2cdmStaticCall σ₀ I otherMessengerCalldata σ σ₁ true o₁ ∧
    L2cdmStaticCall σ₀ I xDomainMessageSenderCalldata σ₁ σ' false o)

/-! ## Success conditions and post-state -/

/-- The conditions under which `expireMessage(H, t)` succeeds, given that the two calls return
    `vOther = otherMessenger()` and `vSender = xDomainMessageSender()`. Arithmetic is on `ℕ`
    (`toNat` of the 256-bit words): the contract's checked `sentAt + P` reverts with
    `Panic(0x11)` if it overflows, so the no-overflow clause is part of the condition. -/
structure ExpireConds (σ : AccountMap) (I : ExecutionEnv) (vOther vSender : AccountAddress) :
    Prop where
  /-- Non-payable: no ETH attached. -/
  noValue : I.weiValue = ⟨0⟩
  /-- ABI: at least 4 + 64 bytes of calldata (and the length is not ≥ 2^255 + 4, where solc's
      signed length check would wrap). -/
  calldataLen : 68 ≤ I.calldata.size ∧ I.calldata.size < 2 ^ 255 + 4
  /-- `msg.sender == L2_CROSS_DOMAIN_MESSENGER`. -/
  callerIsL2cdm : I.source = l2cdm
  /-- `xDomainMessageSender() == otherMessenger()`. -/
  senderIsOther : vSender = vOther
  /-- `sentMessageTimestamps[H] != 0`. -/
  wasSent : sentAt σ I ≠ ⟨0⟩
  /-- `sentAt + EXPIRY_PERIOD` does not overflow 256 bits. -/
  noOverflow : (sentAt σ I).toNat + P_contract < 2 ^ 256
  /-- `_undeliveredAt > sentAt + EXPIRY_PERIOD`. -/
  expired : (sentAt σ I).toNat + P_contract < (argTime I).toNat

/-- The post-state of a successful run, relative to the pre-state `σ`: the persistent storage of
    every account other than the executing contract, the transient storage of every account and
    the code of every account are unchanged, and the contract's persistent storage changed at
    exactly one slot, `expiredMessages[H]`, to `setTrueWord old`. -/
structure ExpirePost (σ σ' : AccountMap) (I : ExecutionEnv) : Prop where
  self_storage :
    (σ'.getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (expiredSlot (argHash I))
        (setTrueWord (storageWord σ I.codeOwner (expiredSlot (argHash I))))
  other_storage : ∀ a, a ≠ I.codeOwner →
    (σ'.getD a default).storage = (σ.getD a default).storage
  tstorage : ∀ a, (σ'.getD a default).tstorage = (σ.getD a default).tstorage
  code : accountCodeStateEq σ σ'

end ExpiryEvm
