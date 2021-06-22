import { ethers } from 'ethers'

export interface EventArgsAddressSet {
  _name: string
  _newAddress: string
  _oldAddress: string
}

export interface EventArgsTransactionEnqueued {
  _l1TxOrigin: string
  _target: string
  _gasLimit: ethers.BigNumber
  _data: string
  _queueIndex: ethers.BigNumber
  _timestamp: ethers.BigNumber
}

export interface EventArgsTransactionBatchAppended {
  _batchIndex: ethers.BigNumber
  _batchRoot: string
  _batchSize: ethers.BigNumber
  _prevTotalElements: ethers.BigNumber
  _extraData: string
}

export interface EventArgsStateBatchAppended {
  _batchIndex: ethers.BigNumber
  _batchRoot: string
  _batchSize: ethers.BigNumber
  _prevTotalElements: ethers.BigNumber
  _extraData: string
}

export interface EventArgsSequencerBatchAppended {
  _startingQueueIndex: ethers.BigNumber
  _numQueueElements: ethers.BigNumber
  _totalElements: ethers.BigNumber
}
