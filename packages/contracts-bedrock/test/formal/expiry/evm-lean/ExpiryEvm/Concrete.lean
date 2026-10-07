import ExpiryEvm.ExpireMessage

/-!
# Concrete runs (reachability witnesses and boundary checks)

EVMLean is executable. These runs execute the pinned bytecode with `Ξ` on concrete states:
the L2ToL2CrossDomainMessenger code at 0x4200..0023, a mock L2CrossDomainMessenger at
0x4200..0007 whose code returns the same address for every call (so
`xDomainMessageSender() == otherMessenger()`), and `sentMessageTimestamps[H] = 5`.
They show the success branch of the theorems is reachable and check the `>` boundary.
They are checked by `native_decide` (compiled evaluation) and are tests, not part of the proof
of the headline theorems.
-/

namespace ExpiryEvm.Concrete

open Ethereum Ethereum.EVM

def messengerAddr : AccountAddress := AccountAddress.ofUInt256 (UInt256.ofNat 0x4200000000000000000000000000000000000023)

/-- `PUSH20 0x..beef PUSH0 MSTORE PUSH1 0x20 PUSH0 RETURN`: returns the word 0x..beef. -/
def mockL2cdmCode : ByteArray :=
  ⟨#[0x73, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xbe, 0xef,
     0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3]⟩

def H : UInt256 := UInt256.ofNat 0x1234

def calldataOf (t : ℕ) : ByteArray :=
  ⟨#[0x76, 0x3a, 0x1c, 0xb7]⟩ ++ UInt256.toByteArray H ++ UInt256.toByteArray (UInt256.ofNat t)

def σ : AccountMap :=
  (∅ : AccountMap)
    |>.insert messengerAddr
      { (default : Account) with
          code := l2tol2Runtime
          storage := (∅ : Storage).insert (sentAtSlot H) (UInt256.ofNat 5) }
    |>.insert l2cdm { (default : Account) with code := mockL2cdmCode }

def env (caller : AccountAddress) (t : ℕ) : ExecutionEnv :=
  { (default : ExecutionEnv) with
      codeOwner := messengerAddr
      sender := caller
      source := caller
      weiValue := ⟨0⟩
      calldata := calldataOf t
      code := l2tol2Runtime
      depth := 1
      perm := true }

def run (caller : AccountAddress) (t : ℕ) :=
  Ξ σ σ (UInt256.ofNat 1000000) default (env caller t)

/-- `expiredMessages[H]` of the messenger after a run, or `none` if the run did not succeed. -/
def expiredAfter (caller : AccountAddress) (t : ℕ) : Option UInt256 :=
  match run caller t with
  | .ok (.success (σ', _, _) _) => some (storageWord σ' messengerAddr (expiredSlot H))
  | _ => none

def reverted (caller : AccountAddress) (t : ℕ) : Bool :=
  match run caller t with
  | .ok (.revert _ _) => true
  | _ => false

#eval expiredAfter l2cdm (5 + P_contract + 1)
#eval reverted l2cdm (5 + P_contract)
#eval reverted (AccountAddress.ofUInt256 (UInt256.ofNat 0x99)) (5 + P_contract + 1)

/-- Reachability witness: at `t = sentAt + P + 1` the call from the L2CrossDomainMessenger
    succeeds and sets `expiredMessages[H]` to 1. -/
theorem success_reachable : expiredAfter l2cdm (5 + P_contract + 1) = some (UInt256.ofNat 1) := by
  native_decide

/-- Boundary: at `t = sentAt + P` (the `≥`-mutation would accept this) the code reverts. -/
theorem boundary_reverts : reverted l2cdm (5 + P_contract) = true := by
  native_decide

/-- Any other caller reverts. -/
theorem other_caller_reverts :
    reverted (AccountAddress.ofUInt256 (UInt256.ofNat 0x99)) (5 + P_contract + 1) = true := by
  native_decide

/-! ## Other branch kinds (witnesses that each disjunct of `expireMessage_outcome` is inhabited) -/

def outcome (σ : AccountMap) (I : ExecutionEnv) (gas : ℕ) : String :=
  match Ξ σ σ (UInt256.ofNat gas) default I with
  | .ok (.success _ _) => "success"
  | .ok (.revert _ _) => "revert"
  | .error .OutOfGass => "oog"
  | .error .StaticModeViolation => "static"
  | .error _ => "other error"

/-- Out of gas: the same successful call with 20000 gas runs out of gas. -/
theorem oog_reachable : outcome σ (env l2cdm (5 + P_contract + 1)) 20000 = "oog" := by
  native_decide

/-- Static mode: entered with `perm = false` (via `STATICCALL`), the run halts at the `SSTORE`. -/
theorem static_reachable :
    outcome σ { env l2cdm (5 + P_contract + 1) with perm := false } 1000000 = "static" := by
  native_decide

/-- A mock L2CrossDomainMessenger whose code is `PUSH0 PUSH0 REVERT`. -/
def σRevertingL2cdm : AccountMap :=
  σ.insert l2cdm { (default : Account) with code := ⟨#[0x5f, 0x5f, 0xfd]⟩ }

/-- Callee failure: if the L2CrossDomainMessenger's view call reverts, `expireMessage` reverts. -/
theorem callee_failure_reverts :
    outcome σRevertingL2cdm (env l2cdm (5 + P_contract + 1)) 1000000 = "revert" := by
  native_decide

/-- `t` of the success witness. -/
def tS : ℕ := 5 + P_contract + 1

/-- The static call `otherMessenger()` from `σ` with 0 forwarded gas. -/
def zeroGasCall :=
  Θ σ σ default (AccountAddress.ofUInt256 (UInt256.ofNat (env l2cdm tS).codeOwner))
    (env l2cdm tS).sender l2cdm (toExecute σ l2cdm) ⟨0⟩
    (UInt256.ofNat (env l2cdm tS).gasPrice) ⟨0⟩ ⟨0⟩ otherMessengerCalldata
    ((env l2cdm tS).depth + 1) (env l2cdm tS).header (env l2cdm tS).blobVersionedHashes
    (env l2cdm tS).blocks false

/-- **Why `CallFailed` is weak.** It holds in `σ`, where the run with `t = sentAt + P + 1`
    succeeds (`success_reachable`): the `otherMessenger()` call with 0 gas fails. Statements
    with a `revert ∧ CallFailed` disjunct therefore do not exclude reverts. -/
theorem callFailed_in_success_state : CallFailed σ σ (env l2cdm tS) := by
  have hz : zeroGasCall.2.2.2.1 = false := by native_decide
  refine Or.inr (Or.inl ⟨zeroGasCall.1, zeroGasCall.2.2.2.2, default, ⟨0⟩, zeroGasCall.2.1,
    zeroGasCall.2.2.1, ?_⟩)
  show _ = zeroGasCall
  rw [← hz]

end ExpiryEvm.Concrete
