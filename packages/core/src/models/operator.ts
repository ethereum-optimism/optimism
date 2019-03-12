export interface EthInfo {
  plasmaChainName: string
}

export interface OperatorTransfer {
  sender: string
  recipient: string
  token: string
  start: string
  end: string
}

export interface OperatorTransaction {
  block: string
  transfers: OperatorTransfer[]
}

export interface OperatorTransferProof {
  parsedSum: string
  leafIndex: string
  signature: string
  inclusionProof: string[]
}

export interface OperatorTransactionProof {
  transferProofs: OperatorTransferProof[]
}

export interface OperatorProof {
  transaction: OperatorTransaction
  transactionProof: OperatorTransactionProof
}

export interface RawOperatorProof {
  deposits: OperatorTransaction[]
  transactionHistory: { [key: number]: OperatorProof[] }
}
