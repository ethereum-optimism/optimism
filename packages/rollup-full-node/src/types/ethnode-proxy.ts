// List of supported web3 rpc methods
export enum Web3RpcMethods {
  getTransactionCount = 'eth_getTransactionCount',
  sendRawTransaction = 'eth_sendRawTransaction',
  call = 'eth_call',
  getTransactionReceipt = 'eth_getTransactionReceipt',
  blockNumber = 'eth_blockNumber',
  gasPrice = 'eth_gasPrice',
  estimateGas = 'eth_estimateGas',
  getCode = 'eth_getCode',
}

// Handler interface which we use to handle incoming requests
export interface Web3RpcHandlerFunctions {
  [Web3RpcMethods.getTransactionCount]: (params: any[]) => Promise<string>
  [Web3RpcMethods.sendRawTransaction]: (params: any[]) => Promise<string>
  [Web3RpcMethods.call]: (params: any[]) => Promise<string>
  [Web3RpcMethods.getTransactionReceipt]: (params: any[]) => Promise<string>
  [Web3RpcMethods.blockNumber]: (params: any[]) => Promise<string>
  [Web3RpcMethods.gasPrice]: (params: any[]) => Promise<string>
  [Web3RpcMethods.estimateGas]: (params: any[]) => Promise<string>
  [Web3RpcMethods.getCode]: (params: any[]) => Promise<string>
}
