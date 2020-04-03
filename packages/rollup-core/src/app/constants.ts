import { ethers } from 'ethers'

export const L1ToL2TransactionEventName = 'L1ToL2Message'
export const L1ToL2TransactionEventId = ethers.utils.id(
  L1ToL2TransactionEventName + '(uint,address,address,bytes)'
)
