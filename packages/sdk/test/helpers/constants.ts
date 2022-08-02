import { ethers } from 'ethers'

export const DUMMY_MESSAGE = {
  target: '0x' + '11'.repeat(20),
  sender: '0x' + '22'.repeat(20),
  message: '0x' + '33'.repeat(64),
  messageNonce: ethers.BigNumber.from(1234),
  value: ethers.BigNumber.from(0),
  minGasLimit: ethers.BigNumber.from(5678),
}

export const DUMMY_EXTENDED_MESSAGE = {
  ...DUMMY_MESSAGE,
  logIndex: 0,
  blockNumber: 1234,
  transactionHash: '0x' + '44'.repeat(32),
}
