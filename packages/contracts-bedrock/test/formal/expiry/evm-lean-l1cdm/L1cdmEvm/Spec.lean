import Reasoning.Solc
import Reasoning.Storage
import L1cdmEvm.Bytecode
import L1cdmEvm.SymMem

/-!
# Statement vocabulary for `L1CrossDomainMessenger.relayUndeliveredMessage`

Solidity source (PR #23259 branch, tip c7c51d79e2):

```solidity
function relayUndeliveredMessage(bytes32 _messageHash, uint256 _undeliveredAt) external {
    if (!systemConfig.isFeatureEnabled(Features.INTEROP)) revert L1CrossDomainMessenger_NotInteropMessenger();
    L1CrossDomainMessenger caller = L1CrossDomainMessenger(msg.sender);
    IOptimismPortal callerPortal = caller.portal();
    if (
        callerPortal.systemConfig().l1CrossDomainMessenger() != msg.sender
            || !portal.ethLockbox().authorizedPortals(callerPortal)
            || caller.xDomainMessageSender() != Predeploys.UNDELIVERED_MESSAGE_EXPORTER
    ) revert L1CrossDomainMessenger_NotInteropMessenger();
    this.sendMessage({
        _target: Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
        _message: abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (_messageHash, _undeliveredAt)),
        _minGasLimit: EXPIRE_MESSAGE_GAS_LIMIT
    });
}
```

Everything is stated on EVMLean's `Ξ`/`Θ`. The words below that come from the compiled code
(`exporterWord`, `expireGasLimit`, selectors, storage slots) are the values the proofs check
against the bytecode; when the branch changes them, change them here and rerun
`scripts/regen.sh` (see README, "Retargeting").
-/

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory L1cdmEvm.SymMem

/-! ## Constants of the compiled artifact -/

/-- `Predeploys.UNDELIVERED_MESSAGE_EXPORTER` as compiled (`PUSH20` at pc 2677). -/
def exporterWord : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000030

/-- `Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER` (`PUSH20` at pc 3068). -/
def l2tol2Word : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000023

/-- `EXPIRE_MESSAGE_GAS_LIMIT` (`PUSH3 0x0186a0` at pc 3090). -/
def expireGasLimit : ℕ := 100000

/-- Storage slots of the L1CrossDomainMessenger (`forge inspect … storageLayout`). -/
def portalSlot : UInt256 := UInt256.ofNat 252
def systemConfigSlot : UInt256 := UInt256.ofNat 254

/-- `bytes4(keccak256("relayUndeliveredMessage(bytes32,uint256)"))`. -/
def relaySelector : UInt256 := UInt256.ofNat 0x372293c3

/-- `bytes32("INTEROP")` (`Features.INTEROP`). -/
def interopWord : UInt256 := UInt256.ofNat 0x494e5445524f5000000000000000000000000000000000000000000000000000

/-- The 2^160 - 1 mask solc applies to addresses. -/
abbrev addrMask : UInt256 := UInt256.ofNat 1461501637330902918203684832716283019655932542975

/-- The address a word names: its low 160 bits (solc masks call targets with `and(w, 2^160-1)`,
    which `AccountAddress.ofUInt256` ignores; `ofUInt256_land_mask`). -/
def addrOf (w : UInt256) : AccountAddress := AccountAddress.ofUInt256 w

/-- The ABI word of an address. -/
def addrWord (a : AccountAddress) : UInt256 := UInt256.ofNat a.val

/-- A word is a clean ABI address (the decoder reverts otherwise). -/
def CleanAddr (w : UInt256) : Prop := w.toNat < 2 ^ 160

/-! ## Calldata view of the outer call -/

abbrev selectorWord (I : ExecutionEnv) : UInt256 := solcSelectorWord I

/-- `_messageHash` = `CALLDATALOAD(4)`. -/
def argHash (I : ExecutionEnv) : UInt256 := uInt256OfByteArray (I.calldata.readBytes 4 32)

/-- `_undeliveredAt` = `CALLDATALOAD(36)`. -/
def argTime (I : ExecutionEnv) : UInt256 := uInt256OfByteArray (I.calldata.readBytes 36 32)

/-- Word at `slot` of account `a`'s storage in `σ` (0 if unset). -/
def storageWord (σ : AccountMap) (a : AccountAddress) (slot : UInt256) : UInt256 :=
  (σ.getD a default).storage.getD slot ⟨0⟩

/-! ## Byte strings -/

/-- 4 selector bytes. -/
def sel4 (s : ℕ) : ByteArray := ⟨#[UInt8.ofNat (s / 2^24), UInt8.ofNat (s / 2^16 % 256),
  UInt8.ofNat (s / 2^8 % 256), UInt8.ofNat (s % 256)]⟩

/-- 32-byte ABI word. -/
abbrev w32 (w : UInt256) : ByteArray := UInt256.toByteArray w

/-! ## Calldata of the external calls -/

/-- `isFeatureEnabled(INTEROP)`. -/
def isFeatureEnabledCd : ByteArray := sel4 0x47af267b ++ w32 interopWord
/-- `portal()`. -/
def portalCd : ByteArray := sel4 0x6425666b
/-- `systemConfig()`. -/
def systemConfigCd : ByteArray := sel4 0x33d7e2bd
/-- `l1CrossDomainMessenger()`. -/
def l1CrossDomainMessengerCd : ByteArray := sel4 0xa7119869
/-- `ethLockbox()`. -/
def ethLockboxCd : ByteArray := sel4 0xb682c444
/-- `authorizedPortals(p)`. -/
def authorizedPortalsCd (p : UInt256) : ByteArray := sel4 0x0fd11077 ++ w32 p
/-- `xDomainMessageSender()`. -/
def xDomainMessageSenderCd : ByteArray := sel4 0x6e296e45

/-- `abi.encodeCall(IL2ToL2CrossDomainMessenger.expireMessage, (H, t))`: 68 bytes. -/
def expireMessageCd (H t : UInt256) : ByteArray := sel4 0x763a1cb7 ++ w32 H ++ w32 t

/-- `abi.encodeCall(this.sendMessage, (0x4200..0023, expireMessage(H, t), 100000))`: 228 bytes:
    selector `0x3dbb202b`, target, offset `0x60`, `_minGasLimit`, length 68, the 68 message bytes,
    28 zero bytes of padding. -/
def sendMessageCd (H t : UInt256) : ByteArray :=
  sel4 0x3dbb202b ++ w32 l2tol2Word ++ w32 (UInt256.ofNat 0x60) ++ w32 (UInt256.ofNat expireGasLimit) ++
    w32 (UInt256.ofNat 68) ++ expireMessageCd H t ++ ByteArray.zeroes 28

/-! ## The deposit the self-call makes

`sendMessage(_target, _message, _minGasLimit)` (in `CrossDomainMessenger`) calls
`portal.depositTransaction{value: msg.value}(otherMessenger, msg.value, baseGas(_message, _minGasLimit),
false, abi.encodeWithSelector(relayMessage.selector, messageNonce(), msg.sender, _target, msg.value,
_minGasLimit, _message))`. For the self-call made by `relayUndeliveredMessage` all of these are fixed
except `otherMessenger`, the nonce, the sender and `H`, `t`. -/

/-- `baseGas(expireMessage(H, t), 100000)` = `21000 + max(200000 + 40000 + 40000 + 5000 +
    100000 * 64 / 63 + (68 + 260) * 16, (68 + 260) * 40)` = 412835. -/
def depositGasLimit : ℕ := 412835

/-- `abi.encodeWithSelector(relayMessage.selector, nonce, sender, 0x4200..0023, 0, 100000,
    expireMessage(H, t))`: 324 bytes. -/
def relayMessageCd (nonce sender H t : UInt256) : ByteArray :=
  sel4 0xd764ad0b ++ w32 nonce ++ w32 sender ++ w32 l2tol2Word ++ w32 (UInt256.ofNat 0) ++
    w32 (UInt256.ofNat expireGasLimit) ++ w32 (UInt256.ofNat 0xc0) ++ w32 (UInt256.ofNat 68) ++
    expireMessageCd H t ++ ByteArray.zeroes 28

/-- `depositTransaction(dest, 0, 412835, false, relayMessageCd …)`: 548 bytes. -/
def depositCd (dest nonce sender H t : UInt256) : ByteArray :=
  sel4 0xe9e05c42 ++ w32 dest ++ w32 (UInt256.ofNat 0) ++ w32 (UInt256.ofNat depositGasLimit) ++
    w32 (UInt256.ofNat 0) ++ w32 (UInt256.ofNat 0xa0) ++ w32 (UInt256.ofNat 324) ++
    relayMessageCd nonce sender H t ++ ByteArray.zeroes 28

/-- `Encoding.encodeVersionedNonce(msgNonce, 1)`: `(1 << 240) | (msgNonce mod 2^240)`. -/
def versionedNonce (msgNonceWord : UInt256) : UInt256 :=
  UInt256.ofNat (2 ^ 240 + msgNonceWord.toNat % 2 ^ 240)

/-- `otherMessenger` (slot 207) of account `a`, masked to an address as the code does. -/
def otherMessengerWord (σ : AccountMap) (a : AccountAddress) : UInt256 :=
  UInt256.land addrMask (storageWord σ a (UInt256.ofNat 207))

/-- `msgNonce` (slot 205) of account `a`. -/
def msgNonceWord (σ : AccountMap) (a : AccountAddress) : UInt256 := storageWord σ a (UInt256.ofNat 205)

/-- `unchecked { ++msgNonce; }` on the slot word `old` (a packed `uint240` write that keeps the top 16
    bits): `(old & ~(2^240 - 1)) | ((old & (2^240 - 1)) + 1) & (2^240 - 1)`. -/
def bumpNonceWord (old : UInt256) : UInt256 :=
  UInt256.lor (UInt256.land old (UInt256.ofNat (2 ^ 256 - 2 ^ 240)))
    (UInt256.land (UInt256.ofNat 1 + UInt256.land old (UInt256.ofNat (2 ^ 240 - 1)))
      (UInt256.ofNat (2 ^ 240 - 1)))

/-- The deposit: a `CALL` from the executing contract to its `portal` (slot 252), value 0, calldata
    `cd`, from account map `σc`, in permission mode `I.perm`; result `σ'`, flag `z`, output `o`. -/
def DepositCall (σ₀ : AccountMap) (I : ExecutionEnv) (cd : ByteArray) (σc σ' : AccountMap) (z : Bool)
    (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in I.codeOwner I.sender
      (addrOf (storageWord σc I.codeOwner portalSlot))
      (toExecute σc (addrOf (storageWord σc I.codeOwner portalSlot))) callGas
      (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd (I.depth + 1) I.header I.blobVersionedHashes I.blocks I.perm

/-! ## External calls and their summaries -/

/-- A static call made by the executing contract (`I.codeOwner`) to `target` with calldata `cd`,
    from account map `σc`: EVMLean's `Θ` with *some* substate and *some* call gas returns account
    map `σ'`, success flag `z` and output `o`. -/
def StaticCallFrom (σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray)
    (σc σ' : AccountMap) (z : Bool) (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in
      (AccountAddress.ofUInt256 (UInt256.ofNat I.codeOwner)) I.sender target
      (toExecute σc target) callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd
      (I.depth + 1) I.header I.blobVersionedHashes I.blocks false

/-- **Summary (hypothesis) of a view call.** From any account map with the storage, transient
    storage and code of `σ`, if the static call to `target` with calldata `cd` *succeeds*, its
    return data is shorter than 2^32 bytes and, if it has at least 32 bytes, its first 32 bytes are
    the ABI word `w`. Nothing is assumed about failing calls or about return data shorter than 32
    bytes (the compiled decoder reverts on those), so `w` is "the value the callee returns". The
    2^32 bound keeps solc's free-memory-pointer arithmetic below 2^256 (real return data is bounded
    by the gas limit; EVMLean's gas model is not used to derive it). -/
def ReturnsWord (σ σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray)
    (w : UInt256) : Prop :=
  ∀ σc σ' z o, accountStorageStateEq σ σc → accountCodeStateEq σ σc →
    StaticCallFrom σ₀ I target cd σc σ' z o → z = true →
      o.size < 2 ^ 32 ∧ (32 ≤ o.size → o.data.toList.take 32 = (w32 w).data.toList)

/-- Successful static calls to `target` with calldata `cd` (from any account map with the storage
    and code of `σ`) return fewer than 2^32 bytes. -/
def CallBound (σ σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray) : Prop :=
  ∀ σc σ' o, accountStorageStateEq σ σc → accountCodeStateEq σ σc →
    StaticCallFrom σ₀ I target cd σc σ' true o → o.size < 2 ^ 32

/-- Some run of the static call to `target` with calldata `cd`, from an account map with the storage
    and code of `σ`, succeeded and returned at least 32 bytes starting with the ABI word `w` (used by
    the trace segments; the actual call of the run is one such). -/
def CallReturned (σ σ₀ : AccountMap) (I : ExecutionEnv) (target : AccountAddress) (cd : ByteArray)
    (w : UInt256) : Prop :=
  ∃ σc σ' o, accountStorageStateEq σ σc ∧ accountCodeStateEq σ σc ∧
    StaticCallFrom σ₀ I target cd σc σ' true o ∧ 32 ≤ o.size ∧
    o.data.toList.take 32 = (w32 w).data.toList

/-- The one state-changing call: `this.sendMessage(...)`, a `CALL` from the executing contract to
    itself (`ADDRESS`), value 0, calldata `cd`, from account map `σc`, in the caller's permission
    mode `I.perm`; result account map `σ'`, flag `z`, output `o`. -/
def SelfCall (σ₀ : AccountMap) (I : ExecutionEnv) (cd : ByteArray) (σc σ' : AccountMap) (z : Bool)
    (o : ByteArray) : Prop :=
  ∃ (A_in : Substate) (callGas g' : UInt256) (A' : Substate),
    (σ', g', A', z, o) = Θ σc σ₀ A_in I.codeOwner I.sender I.codeOwner
      (toExecute σc I.codeOwner) callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd
      (I.depth + 1) I.header I.blobVersionedHashes I.blocks I.perm

/-- The values the seven view calls return (the summaries' `w`s). -/
structure Views where
  feat : UInt256       -- systemConfig.isFeatureEnabled(INTEROP)
  callerPortal : UInt256  -- msg.sender.portal()
  callerSC : UInt256   -- callerPortal.systemConfig()
  callerMsgr : UInt256 -- callerPortal.systemConfig().l1CrossDomainMessenger()
  lockbox : UInt256    -- portal.ethLockbox()
  auth : UInt256       -- portal.ethLockbox().authorizedPortals(callerPortal)
  xSender : UInt256    -- msg.sender.xDomainMessageSender()

/-- The summaries of the seven view calls, with their targets and calldata exactly as the code
    makes them: `systemConfig` and `portal` are this contract's storage slots 254 and 252. -/
structure Summaries (σ σ₀ : AccountMap) (I : ExecutionEnv) (v : Views) : Prop where
  feat : ReturnsWord σ σ₀ I (addrOf (storageWord σ I.codeOwner systemConfigSlot)) isFeatureEnabledCd v.feat
  callerPortal : ReturnsWord σ σ₀ I I.source portalCd v.callerPortal
  callerSC : ReturnsWord σ σ₀ I (addrOf v.callerPortal) systemConfigCd v.callerSC
  callerMsgr : ReturnsWord σ σ₀ I (addrOf v.callerSC) l1CrossDomainMessengerCd v.callerMsgr
  lockbox : ReturnsWord σ σ₀ I (addrOf (storageWord σ I.codeOwner portalSlot)) ethLockboxCd v.lockbox
  auth : ReturnsWord σ σ₀ I (addrOf v.lockbox) (authorizedPortalsCd v.callerPortal) v.auth
  xSender : ReturnsWord σ σ₀ I I.source xDomainMessageSenderCd v.xSender

/-- The conditions under which `relayUndeliveredMessage` can succeed. -/
structure RelayConds (I : ExecutionEnv) (v : Views) : Prop where
  noValue : I.weiValue = ⟨0⟩
  calldataLen : 68 ≤ I.calldata.size ∧ I.calldata.size < 2 ^ 255 + 4
  /-- `systemConfig.isFeatureEnabled(Features.INTEROP)` returned `true`. -/
  interop : v.feat = UInt256.ofNat 1
  /-- the returned addresses are clean (the decoder reverts otherwise). -/
  callerPortalClean : CleanAddr v.callerPortal
  callerSCClean : CleanAddr v.callerSC
  lockboxClean : CleanAddr v.lockbox
  /-- (a) `msg.sender.portal().systemConfig().l1CrossDomainMessenger() == msg.sender`. -/
  isMessenger : v.callerMsgr = addrWord I.source
  /-- (b) `portal.ethLockbox().authorizedPortals(callerPortal)` returned `true`. -/
  authorized : v.auth = UInt256.ofNat 1
  /-- (c) `msg.sender.xDomainMessageSender() == UNDELIVERED_MESSAGE_EXPORTER`. -/
  fromExporter : v.xSender = exporterWord

end L1cdmEvm
