import {
  JsonRpcProvider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/providers'
import { Watcher } from './watcher'
import { Contract, Transaction } from 'ethers'

export const initWatcher = async (
  l1Provider: JsonRpcProvider,
  l2Provider: JsonRpcProvider,
  AddressManager: Contract
) => {
  const l1MessengerAddress = await AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')
  console.log("l1MessengerAddress:",l1MessengerAddress)

  return new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: l1MessengerAddress
    },
    l2: {
      provider: l2Provider,
      messengerAddress: "0x4200000000000000000000000000000000000007"
    },
  })
}


export const initFastWatcher = async (
  l1Provider: JsonRpcProvider,
  l2Provider: JsonRpcProvider,
  AddressManager: Contract,
) => {
  const l1MessengerAddress = await AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessengerFast')
  console.log("l1FastMessengerAddress:",l1MessengerAddress)

  return new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: l1MessengerAddress,
    },
    l2: {
      provider: l2Provider,
      messengerAddress: "0x4200000000000000000000000000000000000007",
    },
  })
}

export interface CrossDomainMessagePair {
  tx: Transaction
  receipt: TransactionReceipt
  remoteReceipt: TransactionReceipt
}

export enum Direction {
  L1ToL2,
  L2ToL1,
}

export const waitForXDomainTransaction = async (
  watcher: Watcher,
  tx: Promise<TransactionResponse> | TransactionResponse,
  direction: Direction,
): Promise<CrossDomainMessagePair> => {

  // await it if needed
  tx = await tx

  // get the receipt and the full transaction
  const receipt = await tx.wait()

  let remoteReceipt: TransactionReceipt

  console.log(' Preparing to wait for Message Hashes')

  if (direction === Direction.L1ToL2) {
    // DEPOSIT
    console.log(' Looking for L1 to L2')
    const [xDomainMsgHash] = await watcher.getMessageHashesFromL1Tx(tx.hash)
    console.log(' Got L1->L2 message hash', xDomainMsgHash)
    remoteReceipt = await watcher.getL2TransactionReceipt(xDomainMsgHash)
    console.log(' Completed Deposit - L2 tx hash:', remoteReceipt.transactionHash)
  } else {
    // WITHDRAWAL
    console.log(' Looking for L2 to L1')
    const [xDomainMsgHash] = await watcher.getMessageHashesFromL2Tx(tx.hash)
    console.log(' Got L2->L1 message hash', xDomainMsgHash)
    remoteReceipt = await watcher.getL1TransactionReceipt(xDomainMsgHash)
    console.log(' Completed Withdrawal - L1 tx hash:', remoteReceipt.transactionHash)
  }

  return {
    tx,
    receipt,
    remoteReceipt,
  }
}
