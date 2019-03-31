export interface StateObject {
  predicate: string
  parameters: any
}

export interface StateUpdate {
  stateId: any
  updateParameters: any
  newState: StateObject
}

export interface Transaction {
  stateUpdate: StateUpdate
  witness: any
  block: number
}

export type TransactionProof = Transaction[]
