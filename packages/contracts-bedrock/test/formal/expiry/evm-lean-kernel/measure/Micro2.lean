import Kernel.KTime
open Ethereum Ethereum.EVM ExpiryEvm
set_option maxHeartbeats 0
set_option maxRecDepth 100000

def codeL : List UInt8 := l2tol2RuntimeFlat.data.toList
/-- direct recursor walk -/
def lenRec (l : List UInt8) : Nat := @List.rec UInt8 (fun _ => Nat) 0 (fun _ _ ih => ih + 1) l
def getRec (l : List UInt8) : Nat → Option UInt8 :=
  @List.rec UInt8 (fun _ => Nat → Option UInt8) (fun _ => none)
    (fun a _ ih n => Nat.rec (motive := fun _ => Option UInt8) (some a) (fun k _ => ih k) n) l
/-- bytes as one big natural number (little endian) -/
def natOfL (l : List UInt8) : Nat := @List.rec UInt8 (fun _ => Nat) 0 (fun a _ ih => a.toNat + 256 * ih) l
def codeN : Nat := natOfL codeL

#ktime codeL.length = 5231
#ktime codeL.length = 5231
#ktime lenRec codeL = 5231
#ktime codeL.lengthTR = 5231
#ktime getRec codeL 4000 = getRec codeL 4000
#ktime codeL[4000]? = codeL[4000]?
#ktime (codeL.drop 4000).head? = (codeL.drop 4000).head?
#ktime codeN % 7 = codeN % 7
#ktime (codeN / 2^(8*4000)) % 256 = (codeN / 2^(8*4000)) % 256
#ktime l2tol2RuntimeFlat[4000]? = l2tol2RuntimeFlat[4000]?
#ktime l2tol2RuntimeFlat.get? 4000 = l2tol2RuntimeFlat.get? 4000
