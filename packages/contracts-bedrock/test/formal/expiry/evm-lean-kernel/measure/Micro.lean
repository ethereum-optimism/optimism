import Kernel.KTime
open Ethereum Ethereum.EVM ExpiryEvm
set_option maxHeartbeats 0
set_option maxRecDepth 100000
#ktime l2tol2RuntimeFlat.size = 5231
#ktime l2tol2RuntimeFlat.data.toList.length = 5231
#ktime l2tol2RuntimeFlat.get! 4000 = 0x5b
#ktime l2tol2RuntimeFlat.get! 10 = 0x00
#ktime parseInstr 0x5b = some .JUMPDEST
#ktime parseInstr 0xfe = some .INVALID
#ktime UInt256.ofNat 4592 = UInt256.ofNat 4592
#ktime (UInt256.ofNat 4592 == UInt256.ofNat 4592) = true
#ktime decode l2tol2RuntimeFlat (UInt256.ofNat 13) = some (.PUSH0, none)
#ktime decode l2tol2RuntimeFlat (UInt256.ofNat 4999) = decode l2tol2RuntimeFlat (UInt256.ofNat 4999)
#ktime decode l2tol2Runtime (UInt256.ofNat 13) = some (.PUSH0, none)
#ktime (#[(⟨87⟩ : UInt256), ⟨124⟩, ⟨183⟩, ⟨217⟩, ⟨221⟩, ⟨232⟩, ⟨241⟩, ⟨251⟩, ⟨260⟩, ⟨271⟩, ⟨282⟩, ⟨293⟩, ⟨302⟩, ⟨339⟩, ⟨350⟩, ⟨358⟩, ⟨377⟩, ⟨388⟩, ⟨449⟩, ⟨462⟩].contains (UInt256.ofNat 462)) = true
