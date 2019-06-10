import BigNum = require('bn.js')

export interface Range {
  start: BigNum
  end: BigNum
}

export interface EcdsaSignature {
  v: string
  r: string
  s: string
}

export interface AbiEncodable {
  encoded: string
}

export interface Transaction {
  plasmaContract: string
  block: number
  range: Range
  methodId: string
  parameters: any
  witness: any
}

export interface StateObject {
  predicateAddress: string,
  data: any
}

export interface StateUpdate {
  stateObject: StateObject,
  range: Range,
  blockNumber: number,
  plasmaContract: string
}