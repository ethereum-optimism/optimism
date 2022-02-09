import {
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/providers'
import { Transaction } from 'ethers'
import { CrossChainMessenger, MessageDirection } from '@eth-optimism/sdk'

export interface CrossDomainMessagePair {
  tx: Transaction
  receipt: TransactionReceipt
  remoteTx: Transaction
  remoteReceipt: TransactionReceipt
}

export enum Direction {
  L1ToL2,
  L2ToL1,
}

export const waitForXDomainTransaction = async (
  messenger: CrossChainMessenger,
  tx: Promise<TransactionResponse> | TransactionResponse
): Promise<CrossDomainMessagePair> => {
  // await it if needed
  tx = await tx

  const receipt = await tx.wait()
  const resolved = await messenger.toCrossChainMessage(tx)
  const messageReceipt = await messenger.waitForMessageReceipt(tx)
  let fullTx: any
  let remoteTx: any
  if (resolved.direction === MessageDirection.L1_TO_L2) {
    fullTx = await messenger.l1Provider.getTransaction(tx.hash)
    remoteTx = await messenger.l2Provider.getTransaction(
      messageReceipt.transactionReceipt.transactionHash
    )
  } else {
    fullTx = await messenger.l2Provider.getTransaction(tx.hash)
    remoteTx = await messenger.l1Provider.getTransaction(
      messageReceipt.transactionReceipt.transactionHash
    )
  }

  return {
    tx: fullTx,
    receipt,
    remoteTx,
    remoteReceipt: messageReceipt.transactionReceipt,
  }
}
