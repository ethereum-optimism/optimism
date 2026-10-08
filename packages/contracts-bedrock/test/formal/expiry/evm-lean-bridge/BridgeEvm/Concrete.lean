import BridgeEvm.Post

/-!
# Concrete runs (reachability witnesses and negative checks)

EVMLean is executable. These runs execute the pinned bridge bytecode with `Ξ` on concrete
states: the bridge code at 0x4200…0024 (run directly, standing in for the proxy's
`DELEGATECALL` frame) with balance `amount`; a mock L2ToL2CrossDomainMessenger at 0x4200…0023
whose code returns the word `1` iff the calldata word at offset 4 equals a fixed hash `Hgood`
(`PUSH32 Hgood PUSH1 4 CALLDATALOAD EQ PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN`), i.e.
`expiredMessages(H) = (H == Hgood)`; and a mock ETHLiquidity at 0x4200…0025 whose code is `STOP`
(so `mint` succeeds and sends nothing; the bridge's own balance funds the SafeSend).

Arguments: `destination = 901`, `nonce = 7`, `from = 0xf0f0`, `to = 0x7070`, `amount = 10^18`.
`Hgood = 0x9fea071a…4d43` was computed independently with foundry:
`cast keccak $(cast abi-encode "f(uint256,uint256,uint256,address,address,bytes)" 901 1 7
0x42…24 0x42…24 $(cast calldata "relayETH(address,address,uint256)" 0xf0f0 0x7070 1e18))`
(chain id 1 = EVMLean's `Ethereum.chainId`). `refundHash_matches_cast` checks that the Lean
definition `refundHash` (the statement's preimage) gives exactly this value.

These are tests checked by `native_decide` (compiled evaluation), not part of the proof of the
headline theorems. They call the implementation directly (`codeOwner` = 0x…24 holding the
implementation's code), so they do not exercise the predeploy `Proxy`'s `DELEGATECALL`; the
theorems cover that frame (any `I.codeOwner`), the tests do not. -/

namespace BridgeEvm.Concrete

open Ethereum Ethereum.EVM

def bridgeAddr : AccountAddress := AccountAddress.ofUInt256 (UInt256.ofNat 0x4200000000000000000000000000000000000024)
def fromAddr : AccountAddress := AccountAddress.ofUInt256 (UInt256.ofNat 0xf0f0)
def amount : ℕ := 1000000000000000000

def Hgood : UInt256 :=
  UInt256.ofNat 0x9fea071a0e10c20158ecc5fbb88e4e051bc1328c9a7c9ecc18b19b9db30d4d43

/-- `PUSH32 Hgood PUSH1 4 CALLDATALOAD EQ PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN`. -/
def mockL2l2Code : ByteArray :=
  ⟨#[0x7f]⟩ ++ UInt256.toByteArray Hgood ++
    ⟨#[0x60, 0x04, 0x35, 0x14, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]⟩

def calldataOf (nonce : ℕ) : ByteArray :=
  ⟨#[0xe1, 0x7a, 0x77, 0x6b]⟩ ++ UInt256.toByteArray (UInt256.ofNat 901) ++
    UInt256.toByteArray (UInt256.ofNat nonce) ++ UInt256.toByteArray (UInt256.ofNat 0xf0f0) ++
    UInt256.toByteArray (UInt256.ofNat 0x7070) ++ UInt256.toByteArray (UInt256.ofNat amount)

def env (nonce : ℕ) : ExecutionEnv :=
  { (default : ExecutionEnv) with
      codeOwner := bridgeAddr
      sender := AccountAddress.ofUInt256 (UInt256.ofNat 0x99)
      source := AccountAddress.ofUInt256 (UInt256.ofNat 0x99)
      weiValue := ⟨0⟩
      calldata := calldataOf nonce
      code := ethbridgeRuntime
      depth := 1
      perm := true }

/-- Pre-state; `refundedPre` is the initial word of `refunded[Hgood]`, `l2l2Code` the messenger
    mock, `liqCode` ETHLiquidity's code, `bal` the bridge's balance. -/
def σx (refundedPre : ℕ) (l2l2Code liqCode : ByteArray) (bal : ℕ) : AccountMap :=
  (∅ : AccountMap)
    |>.insert bridgeAddr
      { (default : Account) with
          code := ethbridgeRuntime
          balance := UInt256.ofNat bal
          storage := if refundedPre = 0 then ∅ else
            (∅ : Storage).insert (refundedSlot Hgood) (UInt256.ofNat refundedPre) }
    |>.insert l2l2 { (default : Account) with code := l2l2Code }
    |>.insert ethLiq { (default : Account) with code := liqCode }

def σ (refundedPre : ℕ) : AccountMap := σx refundedPre mockL2l2Code ⟨#[0x00]⟩ amount

def run (refundedPre nonce : ℕ) :=
  Ξ (σ refundedPre) (σ refundedPre) (UInt256.ofNat 10000000) default (env nonce)

def runx (σ0 : AccountMap) (I : ExecutionEnv) :=
  Ξ σ0 σ0 (UInt256.ofNat 10000000) default I

def revertedx (σ0 : AccountMap) (I : ExecutionEnv) : Bool :=
  match runx σ0 I with
  | .ok (.revert _ _) => true
  | _ => false

def revertSelectorx (σ0 : AccountMap) (I : ExecutionEnv) : Option (List UInt8) :=
  match runx σ0 I with
  | .ok (.revert _ o) => some (o.data.toList.take 4)
  | _ => none

/-- `(refunded[Hgood], balance of from)` after a successful run, `none` otherwise. -/
def afterSuccess (refundedPre nonce : ℕ) : Option (UInt256 × UInt256) :=
  match run refundedPre nonce with
  | .ok (.success (σ', _, _) _) =>
      some (storageWord σ' bridgeAddr (refundedSlot Hgood), (σ'.getD fromAddr default).balance)
  | _ => none

def reverted (refundedPre nonce : ℕ) : Bool :=
  match run refundedPre nonce with
  | .ok (.revert _ _) => true
  | _ => false

/-- The first 4 bytes of the revert data (the custom-error selector), if the run reverted. -/
def revertSelector (refundedPre nonce : ℕ) : Option (List UInt8) :=
  match run refundedPre nonce with
  | .ok (.revert _ o) => some (o.data.toList.take 4)
  | _ => none

/-- `SuperchainETHBridge_MessageNotExpired()` = `0x0978275c` and
    `SuperchainETHBridge_AlreadyRefunded()` = `0x2b792286` (`cast sig`). -/
def selMessageNotExpired : List UInt8 := [0x09, 0x78, 0x27, 0x5c]
def selAlreadyRefunded : List UInt8 := [0x2b, 0x79, 0x22, 0x86]

/-- The statement's preimage definition reproduces Solidity's `abi.encode` hash (cast). -/
theorem refundHash_matches_cast : refundHash (env 7) = Hgood := by native_decide

/-- Reachability witness (success): `refunded[H]` becomes 1 and `from` receives `amount` from
    the SafeSend `SELFDESTRUCT` — the beneficiary of the created contract is `from`. -/
theorem success_reachable :
    afterSuccess 0 7 = some (UInt256.ofNat 1, UInt256.ofNat amount) := by native_decide

/-- Not expired: with `nonce = 8` the recomputed hash is not `Hgood` (wrong preimage), so the
    mock messenger returns false and the code reverts (`SuperchainETHBridge_MessageNotExpired`). -/
theorem wrong_preimage_reverts : revertSelector 0 8 = some selMessageNotExpired := by native_decide

/-- Already refunded: `refunded[Hgood] = 1` makes the code revert (`SuperchainETHBridge_AlreadyRefunded`). -/
theorem already_refunded_reverts : revertSelector 1 7 = some selAlreadyRefunded := by native_decide

/-- A dirty slot whose low byte is 0 still reads `refunded = false` (the success path runs and
    keeps the high bytes: the new word is `old & ~0xff | 1`). -/
theorem dirty_high_bytes_success :
    afterSuccess 0x100 7 = some (UInt256.ofNat 0x101, UInt256.ofNat amount) := by native_decide

/-- Not expired with the right preimage: a messenger mock that returns `0` for every call. -/
theorem not_expired_reverts :
    revertSelectorx (σx 0 ⟨#[0x60, 0x00, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]⟩ ⟨#[0x00]⟩
      amount) (env 7) = some selMessageNotExpired := by native_decide

/-- No code at ETHLiquidity: solc's `EXTCODESIZE` check reverts. -/
theorem no_liquidity_code_reverts : revertedx (σx 0 mockL2l2Code ByteArray.empty amount) (env 7) = true := by
  native_decide

/-- The SafeSend creation fails (the bridge cannot fund `amount`): `CREATE` pushes 0 and the code
    reverts. -/
theorem create_failure_reverts : revertedx (σx 0 mockL2l2Code ⟨#[0x00]⟩ 0) (env 7) = true := by
  native_decide

/-- Entered by `STATICCALL`: the `SSTORE` raises a static-mode violation. -/
theorem static_violation :
    (match runx (σ 0) { env 7 with perm := false } with
     | .error .StaticModeViolation => true | _ => false) = true := by native_decide

/-- In the success run the bridge's storage afterwards has exactly one entry, `refunded[Hgood] = 1`
    (the pre-state storage is empty). This is an observation about this one run (the mock `mint`
    and the SafeSend creation left the bridge's storage alone), not a frame theorem. -/
theorem success_storage_exact :
    (match run 0 7 with
     | .ok (.success (σ', _, _) _) =>
        ((σ'.getD bridgeAddr default).storage.size == 1) &&
          ((σ'.getD bridgeAddr default).storage.getD (refundedSlot Hgood) ⟨0⟩ == UInt256.ofNat 1)
     | _ => false) = true := by native_decide

/-- The hypothesis of `storedMap_post` / `refundETH_store` holds in the concrete pre-state: the
    executing account has non-empty code. -/
theorem bridge_has_code : ((σ 0).getD bridgeAddr default).code.size ≠ 0 := by decide +kernel

end BridgeEvm.Concrete
