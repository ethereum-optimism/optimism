import L1cdmEvm.Spec

/-!
# Concrete runs (reachability witnesses and failing checks)

EVMLean is executable. These run the pinned bytecode with `Ξ` on a concrete world (the same mocks
as `scripts/trace_anvil.py`):

* `SELF` = A's L1CrossDomainMessenger (the code under test): `portal` = `P_A`, `systemConfig` =
  `SC_A`, `otherMessenger` = 0x4200..0007, `msgNonce` = 7;
* a generic mock (`mockCode`: returns the word `SLOAD(selector)`) at `SC_A`, `LB`, `CALLER`
  (B's L1CrossDomainMessenger), `P_B`, `SC_B`, with storage programming the view results (`SC_A`
  answers `isFeatureEnabled(INTEROP)` and `paused()`);
* the portal mock at `P_A` (`portalCode`): like the generic mock, but `depositTransaction` stores
  `keccak256(calldata)` at its slot 0.

They show that the success branch is reachable (and that the deposit's calldata is exactly
`depositCd` of the spec), and that each check failing makes the call revert (a paused chain with
the error `L1CrossDomainMessenger_Paused()`). They are tests
(checked by `native_decide`), not part of the proof of the headline theorems.
-/

namespace L1cdmEvm.Concrete

open Ethereum Ethereum.EVM

def addr (n : ℕ) : AccountAddress := AccountAddress.ofUInt256 (UInt256.ofNat n)
def SELF := addr 0x1010101010101010101010101010101010101010
def SC_A := addr 0x1111111111111111111111111111111111111111
def P_A := addr 0x1212121212121212121212121212121212121212
def LB := addr 0x1313131313131313131313131313131313131313
def CALLER := addr 0x2020202020202020202020202020202020202020
def P_B := addr 0x2121212121212121212121212121212121212121
def SC_B := addr 0x2222222222222222222222222222222222222222

/-- `PUSH0 CALLDATALOAD PUSH1 0xe0 SHR SLOAD PUSH0 MSTORE PUSH1 0x20 PUSH0 RETURN`. -/
def mockCode : ByteArray := ⟨#[0x5f, 0x35, 0x60, 0xe0, 0x1c, 0x54, 0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3]⟩

/-- As `mockCode`, but selector `0xe9e05c42` (`depositTransaction`) jumps to
    `CALLDATACOPY(0, 0, size); SSTORE(0, KECCAK256(0, size)); STOP`. -/
def portalCode : ByteArray :=
  ⟨#[0x5f, 0x35, 0x60, 0xe0, 0x1c, 0x80, 0x63, 0xe9, 0xe0, 0x5c, 0x42, 0x14, 0x60, 0x16, 0x57, 0x54,
     0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3, 0x5b, 0x50, 0x36, 0x5f, 0x5f, 0x37, 0x36, 0x5f, 0x20, 0x5f,
     0x55, 0x00]⟩

def word (a : AccountAddress) : UInt256 := UInt256.ofNat a.val

def store (l : List (ℕ × UInt256)) : Storage :=
  l.foldl (fun s (k, v) => s.insert (UInt256.ofNat k) v) ∅

def acct (c : ByteArray) (l : List (ℕ × UInt256)) : Account :=
  { (default : Account) with code := c, storage := store l, nonce := ⟨1⟩ }

/-- Which check to break. -/
inductive Scenario | ok | noInterop | paused | notMessenger | unauthorized | badSender
  deriving DecidableEq

def world (sc : Scenario) : AccountMap :=
  (∅ : AccountMap)
    |>.insert SELF (acct l1cdmRuntime [(252, word P_A), (254, word SC_A),
        (207, UInt256.ofNat 0x4200000000000000000000000000000000000007), (205, UInt256.ofNat 7),
        (204, UInt256.ofNat 0xdEaD)])
    |>.insert SC_A (acct mockCode [(0x47af267b, UInt256.ofNat (if sc = .noInterop then 0 else 1)),
        (0x5c975abb, UInt256.ofNat (if sc = .paused then 1 else 0))])
    |>.insert CALLER (acct mockCode [(0x6425666b, word P_B),
        (0x6e296e45, if sc = .badSender then UInt256.ofNat 0x99 else exporterWord)])
    |>.insert P_B (acct mockCode [(0x33d7e2bd, word SC_B)])
    |>.insert SC_B (acct mockCode [(0xa7119869, if sc = .notMessenger then UInt256.ofNat 0x98 else word CALLER)])
    |>.insert P_A (acct portalCode [(0xb682c444, word LB)])
    |>.insert LB (acct mockCode [(0x0fd11077, UInt256.ofNat (if sc = .unauthorized then 0 else 1))])

def H : UInt256 := UInt256.ofNat 0x1234
def T : UInt256 := UInt256.ofNat 1000000000

def env : ExecutionEnv :=
  { (default : ExecutionEnv) with
      codeOwner := SELF
      sender := CALLER
      source := CALLER
      weiValue := ⟨0⟩
      calldata := sel4 0x372293c3 ++ w32 H ++ w32 T
      code := l1cdmRuntime
      depth := 1
      perm := true }

def run (sc : Scenario) := Ξ (world sc) (world sc) (UInt256.ofNat 3000000) default env

def storageAt (σ : AccountMap) (a : AccountAddress) (k : ℕ) : UInt256 :=
  (σ.getD a default).storage.getD (UInt256.ofNat k) ⟨0⟩

/-- The deposit calldata the spec predicts for this world: `otherMessenger` = 0x..07, nonce
    `(1 << 240) | 7`, sender = `SELF` (the self-call's `msg.sender`). -/
def expectedDeposit : ByteArray :=
  depositCd (UInt256.ofNat 0x4200000000000000000000000000000000000007) (versionedNonce (UInt256.ofNat 7))
    (word SELF) H T

/-- On success: (portal's recorded keccak of the deposit calldata, A's msgNonce afterwards). -/
def successFacts (sc : Scenario) : Option (UInt256 × UInt256) :=
  match run sc with
  | .ok (.success (σ', _, _) _) => some (storageAt σ' P_A 0, storageAt σ' SELF 205)
  | _ => none

def reverted (sc : Scenario) : Bool :=
  match run sc with
  | .ok (.revert _ _) => true
  | _ => false

/-- The revert data of a reverting run. -/
def revertData (sc : Scenario) : Option (List UInt8) :=
  match run sc with
  | .ok (.revert _ o) => some o.data.toList
  | _ => none

def keccakWord (b : ByteArray) : UInt256 := UInt256.ofNat (fromByteArrayBigEndian (KEC b))

/-- **Reachability.** With all checks passing, the call succeeds; the portal received exactly
    `depositCd` (its keccak matches) and `msgNonce` went from 7 to 8. -/
theorem success_reachable :
    successFacts .ok = some (keccakWord expectedDeposit, UInt256.ofNat 8) := by
  native_decide

theorem noInterop_reverts : reverted .noInterop = true := by native_decide

/-- **Paused.** With `paused()` answering `true` (and every other check passing), the call
    reverts with the error `L1CrossDomainMessenger_Paused()` (selector `0xfee42a06`). -/
theorem paused_reverts : revertData .paused = some [0xfe, 0xe4, 0x2a, 0x06] := by native_decide
theorem notMessenger_reverts : reverted .notMessenger = true := by native_decide
theorem unauthorized_reverts : reverted .unauthorized = true := by native_decide
theorem badSender_reverts : reverted .badSender = true := by native_decide

/-- A caller that is not a messenger at all (an EOA-like account with no code) also reverts. -/
theorem other_caller_reverts :
    (match Ξ (world .ok) (world .ok) (UInt256.ofNat 3000000) default
        { env with source := addr 0x99, sender := addr 0x99 } with
      | .ok (.revert _ _) => true
      | _ => false) = true := by
  native_decide

end L1cdmEvm.Concrete
