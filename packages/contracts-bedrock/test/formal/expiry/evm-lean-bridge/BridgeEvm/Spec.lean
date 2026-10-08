import Reasoning.Solc
import Reasoning.Storage
import BridgeEvm.Bytecode
import BridgeEvm.Mem

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-- `bytes4(keccak256("refundETH(uint256,uint256,address,address,uint256)"))`. -/
def refundSelector : UInt256 := UInt256.ofNat 0xe17a776b

abbrev selectorWord (I : ExecutionEnv) : UInt256 := solcSelectorWord I

/-- The calldata word at byte offset `off` (`CALLDATALOAD(off)`). -/
def argWord (I : ExecutionEnv) (off : ℕ) : UInt256 := uInt256OfByteArray (I.calldata.readBytes off 32)

def argDest (I : ExecutionEnv) : UInt256 := argWord I 4
def argNonce (I : ExecutionEnv) : UInt256 := argWord I 36
def argFrom (I : ExecutionEnv) : UInt256 := argWord I 68
def argTo (I : ExecutionEnv) : UInt256 := argWord I 100
def argAmount (I : ExecutionEnv) : UInt256 := argWord I 132

/-- `type(uint160).max`, solc's address mask. -/
def addrMask : UInt256 := UInt256.ofNat 1461501637330902918203684832716283019655932542975

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-- The ABI-level conditions of a call to `refundETH`: no ETH attached (non-payable), at least
    `4 + 5·32 = 164` bytes of calldata (and fewer than `2^255 + 4`, where solc's signed length
    check would wrap), and the two `address` arguments are clean (high 96 bits zero; solc's
    validator reverts otherwise). -/
structure ArgsOk (I : ExecutionEnv) : Prop where
  noValue : I.weiValue = ⟨0⟩
  calldataLen : 164 ≤ I.calldata.size ∧ I.calldata.size < 2 ^ 255 + 4
  fromClean : (argFrom I).toNat < 2 ^ 160
  toClean : (argTo I).toNat < 2 ^ 160

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-! ## The message hash `refundETH` recomputes

Solidity (`Hashing.hashL2toL2CrossDomainMessage`):
`keccak256(abi.encode(_destination, block.chainid, _nonce, address(this), address(this),
abi.encodeCall(this.relayETH, (_from, _to, _amount))))`. -/

/-- `address(this)` as a word: the executing account (`ADDRESS`), i.e. the bridge proxy when the
    implementation runs under `DELEGATECALL`. -/
def selfWord (I : ExecutionEnv) : UInt256 := UInt256.ofNat I.codeOwner.val

/-- `block.chainid` as EVMLean's `CHAINID` returns it: the constant `Ethereum.chainId` (EVMLean
    fixes the chain id; it is not part of the execution environment). -/
def chainIdWord : UInt256 := UInt256.ofNat Ethereum.chainId

/-- `bytes4(keccak256("relayETH(address,address,uint256)"))` = `0x4f0edcc9`. -/
def relayETHSelector : ByteArray := ⟨#[0x4f, 0x0e, 0xdc, 0xc9]⟩

/-- **The exact 352-byte preimage** of the message hash, as the ABI defines `abi.encode` of
    `(uint256, uint256, uint256, address, address, bytes)` with
    `bytes = relayETH.selector ‖ abi.encode(from, to, amount)` (100 bytes):

    | bytes      | content                                    |
    |------------|--------------------------------------------|
    | 0..32      | `_destination`                             |
    | 32..64     | `block.chainid`                            |
    | 64..96     | `_nonce`                                   |
    | 96..128    | `address(this)` (sender)                   |
    | 128..160   | `address(this)` (target)                   |
    | 160..192   | `0xc0` (offset of the `bytes` tail)        |
    | 192..224   | `100` (length of the `bytes`)              |
    | 224..228   | `0x4f0edcc9` (`relayETH` selector)          |
    | 228..260   | `_from`                                    |
    | 260..292   | `_to`                                      |
    | 292..324   | `_amount`                                  |
    | 324..352   | 28 zero bytes (padding to a word)          |
-/
def refundPreimage (I : ExecutionEnv) : ByteArray :=
  UInt256.toByteArray (argDest I) ++ UInt256.toByteArray chainIdWord ++
    UInt256.toByteArray (argNonce I) ++ UInt256.toByteArray (selfWord I) ++
    UInt256.toByteArray (selfWord I) ++ UInt256.toByteArray (UInt256.ofNat 0xc0) ++
    UInt256.toByteArray (UInt256.ofNat 100) ++ relayETHSelector ++
    UInt256.toByteArray (argFrom I) ++ UInt256.toByteArray (argTo I) ++
    UInt256.toByteArray (argAmount I) ++ ByteArray.zeroes 28

/-- `H`: the message hash, `keccak256(refundPreimage)` as a word (EVMLean's `KEC`). -/
def refundHash (I : ExecutionEnv) : UInt256 :=
  UInt256.ofNat (fromByteArrayBigEndian (KEC (refundPreimage I)))

/-- `0x4f0edcc9 << 224`, the word solc merges the selector from. -/
abbrev relaySelWord : UInt256 :=
  UInt256.ofNat 35758974700130516083418609194394918359861923750008682890162270758914934964224

/-- The preimage as a byte list, in the normal form the trace produces. -/
abbrev refundPreimageL (I : ExecutionEnv) : List UInt8 :=
  wb (argDest I) ++ (wb chainIdWord ++ (wb (argNonce I) ++ (wb (selfWord I) ++ (wb (selfWord I) ++
    (wb (UInt256.ofNat 192) ++ (wb (UInt256.ofNat 100) ++ ((wb relaySelWord).take 4 ++
    (wb (argFrom I) ++ (wb (argTo I) ++ (wb (argAmount I) ++ (wb (UInt256.ofNat 0)).take 28))))))))))

theorem relayETHSelector_eq : relayETHSelector = ofL ((wb relaySelWord).take 4) := by
  decide +kernel

theorem zeroes28_eq : ByteArray.zeroes 28 = ofL ((wb (UInt256.ofNat 0)).take 28) := by
  decide +kernel

theorem refundPreimage_eq (I : ExecutionEnv) : refundPreimage I = ofL (refundPreimageL I) := by
  unfold refundPreimage
  rw [relayETHSelector_eq, zeroes28_eq]
  simp only [toByteArray_eq_ofL, append_ofL, List.append_assoc]

theorem refundPreimage_size (I : ExecutionEnv) : (refundPreimage I).size = 352 := by
  rw [refundPreimage_eq]; simp

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-! ## External calls -/

/-- `Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER` = 0x4200…0023 as a word / address. -/
def l2l2Word : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000023
def l2l2 : AccountAddress := AccountAddress.ofUInt256 l2l2Word

/-- `expiredMessages(bytes32)` selector `0xd5a522fe` (the public mapping's getter). -/
def expiredMessagesSelector : ByteArray := ⟨#[0xd5, 0xa5, 0x22, 0xfe]⟩

/-- Calldata of `expiredMessages(H)`: selector ‖ `H` (36 bytes). -/
def expiredCalldata (H : UInt256) : ByteArray := expiredMessagesSelector ++ UInt256.toByteArray H

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

/-- The first 32 bytes of a call's return data as a word (what solc's ABI decoder reads). -/
def returnWord (o : ByteArray) : UInt256 := UInt256.ofNat (fromBytesBigEndian (o.data.toList.take 32))

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

abbrev expiredSelWord : UInt256 :=
  UInt256.ofNat 96634408021126418473330570835253591908004151213357964794993293693437865885696

theorem expiredCalldata_eq (H : UInt256) :
    expiredCalldata H = ofL ((wb expiredSelWord).take 4 ++ wb H) := by
  unfold expiredCalldata
  rw [show expiredMessagesSelector = ofL ((wb expiredSelWord).take 4) by decide +kernel,
    toByteArray_eq_ofL, append_ofL]

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-! ## Storage: `mapping(bytes32 => bool) public refunded` at slot 0 -/

/-- Storage slot of `refunded[H]`: `keccak256(H ‖ 0)`. -/
def refundedSlot (H : UInt256) : UInt256 := solcMappingSlot (UInt256.ofNat 0) H

/-- Word at `slot` of account `a`'s persistent storage in `σ` (0 if unset or no account). -/
def storageWord (σ : AccountMap) (a : AccountAddress) (slot : UInt256) : UInt256 :=
  (σ.getD a default).storage.getD slot ⟨0⟩

/-- The storage word holding `refunded[H]` in the executing account. Solidity reads the bool as
    its low byte: `refunded[H]` is false iff `refundedWord σ I H & 0xff = 0`. -/
def refundedWord (σ : AccountMap) (I : ExecutionEnv) (H : UInt256) : UInt256 :=
  storageWord σ I.codeOwner (refundedSlot H)

/-- Solidity's packed-bool write of `true`: `(old & ~0xff) | 1` (other 31 bytes kept). -/
def setTrueWord (old : UInt256) : UInt256 :=
  UInt256.lor (UInt256.ofNat 1)
    (UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680) old)

/-- `Predeploys.ETH_LIQUIDITY` = 0x4200…0025. -/
def ethLiqWord : UInt256 := UInt256.ofNat 0x4200000000000000000000000000000000000025
def ethLiq : AccountAddress := AccountAddress.ofUInt256 ethLiqWord

/-- `mint(uint256)` selector `0xa0712d68`. -/
def mintSelector : ByteArray := ⟨#[0xa0, 0x71, 0x2d, 0x68]⟩

/-- Calldata of `ETHLiquidity.mint(amount)` (36 bytes). -/
def mintCalldata (amount : UInt256) : ByteArray := mintSelector ++ UInt256.toByteArray amount

abbrev mintSelWord : UInt256 :=
  UInt256.ofNat 72570022874062638528011751457397263716769196454539065078543251854057308946432

theorem mintCalldata_eq (a : UInt256) : mintCalldata a = ofL ((wb mintSelWord).take 4 ++ wb a) := by
  unfold mintCalldata
  rw [show mintSelector = ofL ((wb mintSelWord).take 4) by decide +kernel, toByteArray_eq_ofL,
    append_ofL]

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

end BridgeEvm

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Mem

/-! ## `new SafeSend{value: amount}(payable(from))` -/

/-- SafeSend's creation code as compiled (89 bytes): forge's `SafeSend.bytecode.object` for the
    pinned commit (`keccak256 = 0xfd5b2655…` — see README; `scripts/regen.sh` checks it is embedded
    in the bridge runtime at offset 3030, the `CODECOPY` source of `new SafeSend`). Its constructor
    ABI-decodes one `address` from the end of the init code and executes
    `SELFDESTRUCT(_recipient)`. -/
def safeSendInitcode : ByteArray :=
  ⟨#[0x60, 0x80, 0x60, 0x40, 0x52, 0x60, 0x40, 0x51, 0x60, 0x59, 0x38, 0x03, 0x80, 0x60, 0x59, 0x83,
    0x39, 0x81, 0x01, 0x60, 0x40, 0x81, 0x90, 0x52, 0x60, 0x1e, 0x91, 0x60, 0x2a, 0x56, 0x5b, 0x80,
    0x60, 0x01, 0x60, 0x01, 0x60, 0xa0, 0x1b, 0x03, 0x16, 0xff, 0x5b, 0x60, 0x00, 0x60, 0x20, 0x82,
    0x84, 0x03, 0x12, 0x15, 0x60, 0x3b, 0x57, 0x60, 0x00, 0x80, 0xfd, 0x5b, 0x81, 0x51, 0x60, 0x01,
    0x60, 0x01, 0x60, 0xa0, 0x1b, 0x03, 0x81, 0x16, 0x81, 0x14, 0x60, 0x51, 0x57, 0x60, 0x00, 0x80,
    0xfd, 0x5b, 0x93, 0x92, 0x50, 0x50, 0x50, 0x56, 0xfe]⟩

/-- The init code `refundETH` passes to `CREATE`: SafeSend's creation code followed by the ABI
    encoding of the constructor argument `payable(_from)` (121 bytes). -/
def safeSendDeploy (recipient : UInt256) : ByteArray := safeSendInitcode ++ UInt256.toByteArray recipient

end BridgeEvm
