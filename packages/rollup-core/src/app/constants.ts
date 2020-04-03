import { ethers } from 'ethers'

export const L1ToL2TransactionEventId = ethers.utils.id(
  'L1ToL2Message(uint,address,address,bytes'
)
