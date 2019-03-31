export interface EventFilter {
  event: string
  address: string
  abi: any
  fromBlock: number
  toBlock: number
  indexed?: { [key: string]: any }
}

export interface EventLog {
  address: string
  logIndex: number
  transactionIndex: number
  transactionHash: string
  blockHash: string
  blockNumber: number
}

export interface TransactionReceipt {
  status: boolean
  blockHash: string
  blockNumber: number
  transactionHash: string
  transactionIndex: number
  from: string
  to: string
  contractAddress?: string
  cumulativeGasUsed: number
  gasUsed: number
  logs: EventLog[]
}

export interface Account {
  address: string
  privateKey: string
}
