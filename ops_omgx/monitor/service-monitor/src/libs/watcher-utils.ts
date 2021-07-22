import {
  JsonRpcProvider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/providers'
import { Watcher } from './watcher'
import { Contract, Transaction } from 'ethers'
import logger from '../logger'

export const initWatcher = async (
  l1Provider: JsonRpcProvider,
  l2Provider: JsonRpcProvider,
  AddressManager: Contract
) => {

  // const l1MessengerAddress = '0xF10EEfC14eB5b7885Ea9F7A631a21c7a82cf5D76'
  const l1MessengerAddress = await AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessenger')
  logger.info('l1MessengerAddress: ' + l1MessengerAddress)

  return new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: l1MessengerAddress,
    },
    l2: {
      provider: l2Provider,
      messengerAddress: '0x4200000000000000000000000000000000000007',
    },
  })
}

export const initFastWatcher = async (
  l1Provider: JsonRpcProvider,
  l2Provider: JsonRpcProvider,
  AddressManager: Contract,
) => {

  // const l1MessengerAddress = '0xF296F4ca6A5725F55EdF1C67F80204871E65F87d'
  const l1MessengerAddress = await AddressManager.getAddress('Proxy__OVM_L1CrossDomainMessengerFast')
  logger.info('l1FastMessengerAddress: ' + l1MessengerAddress)

  return new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: l1MessengerAddress,
    },
    l2: {
      provider: l2Provider,
      messengerAddress: '0x4200000000000000000000000000000000000007',
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
  direction: Direction
): Promise<CrossDomainMessagePair> => {

  // await it if needed
  tx = await tx

  // get the receipt and the full transaction
  const receipt = await tx.wait()

  let remoteReceipt: TransactionReceipt
  if (direction === Direction.L1ToL2) {
    // DEPOSIT
    const [xDomainMsgHash] = await watcher.getMessageHashesFromL1Tx(tx.hash)
    logger.info(' Got L1->L2 message hash: ' + xDomainMsgHash)
    remoteReceipt = await watcher.getL2TransactionReceipt(xDomainMsgHash)
    logger.info(' Completed Deposit! L2 tx hash: ' + remoteReceipt.transactionHash)
  } else {
    // WITHDRAWAL
    const [xDomainMsgHash] = await watcher.getMessageHashesFromL2Tx(tx.hash)
    logger.info(' Got L2->L1 message hash: ' + xDomainMsgHash)
    remoteReceipt = await watcher.getL1TransactionReceipt(xDomainMsgHash)
    logger.info(' Completed Withdrawal! L1 tx hash: ' + remoteReceipt.transactionHash)
  }

  return {
    tx,
    receipt,
    remoteReceipt,
  }
}
