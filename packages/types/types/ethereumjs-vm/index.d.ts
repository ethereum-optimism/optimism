declare namespace EthereumjsVM {
  interface VMError {
    error: string
    errorType: string
  }

  interface VMState {
    runState: any
    exception: number
    exceptionError: VMError
    logs: any[]
    selfdestruct: { [key: string]: any }
    return: Buffer
  }

  interface ExecutionResult {
    gas: any
    gasUsed: any
    gasRefund: any
    createdAddress?: Buffer
    vm: VMState
  }

  interface TxExecutionOptions {
    tx: any
    skipBalance?: boolean
    skipNonce?: boolean
  }

  interface VMOptions {
    enableHomestead?: boolean
    activatePrecompiles?: boolean
  }

  interface GenesisData {
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
}

declare module 'ethereumjs-vm' {
  export = EthereumjsVM.VM
}
