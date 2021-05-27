import {
  JsonRpcProvider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/providers'
import { Watcher } from '@eth-optimism/core-utils'

import { Contract, Transaction } from 'ethers'
import { ethers } from 'ethers'

export const initWatcher = async (
  l1Provider: JsonRpcProvider,
  l2Provider: JsonRpcProvider,
  AddressManager: Contract
) => {
  const l1MessengerAddress = await AddressManager.getAddress(
    'Proxy__OVM_L1CrossDomainMessenger'
  )
  const l2MessengerAddress = await AddressManager.getAddress(
    'OVM_L2CrossDomainMessenger'
  )
  return new Watcher({
    l1: {
      provider: l1Provider,
      messengerAddress: l1MessengerAddress,
    },
    l2: {
      provider: l2Provider,
      messengerAddress: l2MessengerAddress,
    },
  })
}

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
  watcher: Watcher,
  tx: Promise<TransactionResponse> | TransactionResponse,
  direction: Direction
): Promise<CrossDomainMessagePair> => {
  const { src, dest } =
    direction === Direction.L1ToL2
      ? { src: watcher.l1, dest: watcher.l2 }
      : { src: watcher.l2, dest: watcher.l1 }
  // await it if needed
  tx = await tx
  // get the receipt and the full transaction
  const receipt = await tx.wait()
  const fullTx = await src.provider.getTransaction(tx.hash)
  const sleep=async (ms: number) => {
      return new Promise((resolve) => {
          setTimeout(() => {
              resolve('');
          }, ms)
      });
  }
  // get the message hash which was created on the SentMessage
  //const [xDomainMsgHash] = await watcher.getMessageHashesFromTx(src, tx.hash)
  const receipt2 = await src.provider.getTransactionReceipt(tx.hash)
  const msgHashes = []
  for (const log of receipt2.logs) {
    if (
      log.topics[0] === ethers.utils.id('SentMessage(bytes)')
    ) {
      const [message] = ethers.utils.defaultAbiCoder.decode(
          ['bytes'],
          log.data
        )
      msgHashes.push(ethers.utils.solidityKeccak256(['bytes'],[message]))
    }
  }
  const [xDomainMsgHash] = msgHashes
  // Get the transaction and receipt on the remote layer
  // const remoteReceipt = await watcher.getTransactionReceipt(
  //   dest,
  //   xDomainMsgHash
  // )
  
  await sleep(5000)
  const blockNumber = await dest.provider.getBlockNumber()
  const startingBlock = Math.max(blockNumber - 10, 0)
  const filter = {
    address: dest.messengerAddress,
    topics: [],
    fromBlock: startingBlock,
  }
  const logs = await dest.provider.getLogs(filter)
  const matches = logs.filter((log: any) => {
    log.data === xDomainMsgHash})
  var remoteReceipt = null
  // Message was relayed in the past
  if (matches.length > 0) {
    if (matches.length > 1) {
      throw Error(
        'Found multiple transactions relaying the same message hash.'
      )
    }
    remoteReceipt = dest.provider.getTransactionReceipt(matches[0].transactionHash)
  }
  
  if(remoteReceipt==null){
    // Message has yet to be relayed, poll until it is found
    remoteReceipt=await new Promise(async (resolve, reject) => {
      dest.provider.on(filter, async (log: any) => {
        if (log.data === xDomainMsgHash) {
          try {
            const txReceipt = await dest.provider.getTransactionReceipt(
              log.transactionHash
            )
            dest.provider.off(filter)
            resolve(txReceipt)
          } catch (e) {
            reject(e)
          }
        }
      })
    })
  }
    
  console.log(ethers.utils.id('DepositFinalized(address indexed,uint256)'),remoteReceipt.logs)
    
  const remoteTx = await dest.provider.getTransaction(
    remoteReceipt.transactionHash
  )
  return {
    tx: fullTx,
    receipt,
    remoteTx,
    remoteReceipt,
  }
}
