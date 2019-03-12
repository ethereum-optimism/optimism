declare module 'ethereumjs-vm' {
  import BigNum from 'bn.js'
  import EthereumTx from 'ethereumjs-tx'

  interface VMError {
    error: string
    errorType: string
  }

  interface VMState {
    runState: any
    exception: number
    exceptionError: VMError
    logs: any[]
    selfdestruct: { [key: string]: BigNum }
    return: Buffer
  }

  export interface ExecutionResult {
    gas: BigNum
    gasUsed: BigNum
    gasRefund: BigNum
    createdAddress?: Buffer
    vm: VMState
  }

  export interface TxExecutionOptions {
    tx: EthereumTx
    skipBalance?: boolean
    skipNonce?: boolean
  }

  export interface VMOptions {
    enableHomestead?: boolean
    activatePrecompiles?: boolean
  }

  export interface GenesisData {
    [key: string]: string
  }

  class StateManager {
    public generateGenesis(
      initState: GenesisData,
      callback?: (err?: Error, result?: void) => void
    ): void
  }

  class VM {
    public stateManager: StateManager
    constructor(options?: VMOptions)
    public runTx(
      options: TxExecutionOptions,
      callback?: (err?: Error, result?: ExecutionResult) => void
    ): ExecutionResult
  }

  export default VM
}
