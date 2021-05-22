/* Imports: External */
import { ethers } from 'ethers'

export type TypedEthersEvent<T> = ethers.Event & {
  args: T
}

export interface EventArgsAddressSet {
  _name: string
  _newAddress: string
}

export interface EventArgsTransactionEnqueued {
  _chainId: ethers.BigNumber
  _l1TxOrigin: string
  _target: string
  _gasLimit: ethers.BigNumber
  _data: string
  _queueIndex: ethers.BigNumber
  _timestamp: ethers.BigNumber
}

export interface EventArgsTransactionBatchAppended {
  _chainId: ethers.BigNumber
  _batchIndex: ethers.BigNumber
  _batchRoot: string
  _batchSize: ethers.BigNumber
  _prevTotalElements: ethers.BigNumber
  _extraData: string
}

export interface EventArgsStateBatchAppended {
  _chainId: ethers.BigNumber
  _batchIndex: ethers.BigNumber
  _batchRoot: string
  _batchSize: ethers.BigNumber
  _prevTotalElements: ethers.BigNumber
  _extraData: string
}

export interface EventArgsSequencerBatchAppended {
  _chainId: ethers.BigNumber
  _startingQueueIndex: ethers.BigNumber
  _numQueueElements: ethers.BigNumber
  _totalElements: ethers.BigNumber
}
