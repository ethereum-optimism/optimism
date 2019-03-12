import EVM, {
  ExecutionResult,
  GenesisData,
  TxExecutionOptions,
  VMOptions,
} from 'ethereumjs-vm'

export class VM {
  private vm: EVM

  constructor(options?: VMOptions) {
    this.vm = new EVM(options)
  }

  public generateGenesis(initState: GenesisData): Promise<any> {
    return new Promise<void>((resolve, reject) => {
      this.vm.stateManager.generateGenesis(initState, (err, result) => {
        if (err) {
          reject(err)
        }
        resolve(result)
      })
    })
  }

  public runTx(options: TxExecutionOptions): Promise<ExecutionResult> {
    return new Promise<ExecutionResult>((resolve, reject) => {
      this.vm.runTx(options, (err, result) => {
        if (err) {
          reject(err)
        }
        resolve(result)
      })
    })
  }
}
