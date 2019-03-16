import VM = require('ethereumjs-vm')

export class EVM {
  private vm: VM

  constructor(options?: EthereumjsVM.VMOptions) {
    this.vm = new VM(options)
  }

  public generateGenesis(initState: EthereumjsVM.GenesisData): Promise<any> {
    return new Promise<void>((resolve, reject) => {
      this.vm.stateManager.generateGenesis(initState, (err, result) => {
        if (err) {
          reject(err)
        }
        resolve(result)
      })
    })
  }

  public runTx(
    options: EthereumjsVM.TxExecutionOptions
  ): Promise<EthereumjsVM.ExecutionResult> {
    return new Promise<EthereumjsVM.ExecutionResult>((resolve, reject) => {
      this.vm.runTx(options, (err, result) => {
        if (err) {
          reject(err)
        }
        resolve(result)
      })
    })
  }
}
