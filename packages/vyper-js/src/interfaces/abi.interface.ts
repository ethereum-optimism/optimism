export interface VyperAbiItem {
  type: string
  name: string
  indexed?: boolean
}

export interface VyperAbiMethod {
  name: string
  type: string
  inputs?: VyperAbiItem[]
  output?: VyperAbiItem[]
  anonymous?: boolean
  constant?: boolean
  payable?: boolean
  gas?: number
}
