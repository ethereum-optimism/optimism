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

end ExpiryEvm.Concrete
