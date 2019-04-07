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

export interface ContractFunctionArguments {
  params?: any[]
  options?: ContractFunctionOptions
}

export interface ContractFunctionOptions {
  from?: string
  gasPrice?: string
  gas?: number
  value?: number
}

export type ContractFunction = (args?: ContractFunctionArguments) => any

export interface Contract {
  readonly address: string
  readonly abi: Abi
  readonly methods: Record<string, ContractFunction>
}

export type AbiType = 'function' | 'constructor' | 'event' | 'fallback'
export type StateMutabilityType = 'pure' | 'view' | 'nonpayable' | 'payable'

export type Abi = AbiItem | AbiItem[]

export interface AbiItem {
  anonymous?: boolean
  constant?: boolean
  inputs?: AbiInput[]
  name?: string
  outputs?: AbiOutput[]
  payable?: boolean
  stateMutability?: StateMutabilityType
  type: AbiType
}

export interface AbiInput {
  name: string
  type: string
  indexed?: boolean
  components?: AbiInput[]
}

export interface AbiOutput {
  name: string
  type: string
  components?: AbiOutput[]
}
