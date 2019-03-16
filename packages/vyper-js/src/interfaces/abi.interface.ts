export type VyperAbiType = 'function' | 'constructor' | 'event' | 'fallback'
export type VyperStateMutabilityType =
  | 'pure'
  | 'view'
  | 'nonpayable'
  | 'payable'

export interface VyperAbiInput {
  type: string
  name: string
  indexed?: boolean
}

export interface VyperAbiOutput {
  type: string
  name: string
}

export interface VyperAbiMethod {
  name: string
  type: VyperAbiType
  inputs?: VyperAbiInput[]
  output?: VyperAbiOutput[]
  anonymous?: boolean
  constant?: boolean
  payable?: boolean
  gas?: number
  stateMutability?: VyperStateMutabilityType
}
