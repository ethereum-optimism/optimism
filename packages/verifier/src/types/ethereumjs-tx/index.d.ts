declare module 'ethereumjs-tx' {
  interface EthereumTxParams {
    nonce: string
    gasPrice: string
    gasLimit: string
    from?: string
    to?: string
    value: string
    data: string
  }

  class EthereumTx {
    constructor(args: EthereumTxParams)
    public sign(privateKey: Buffer): void
  }

  export = EthereumTx
}
