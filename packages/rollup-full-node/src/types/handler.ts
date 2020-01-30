// Web3 handler interface
import { Address } from '@pigi/rollup-core'

export interface FullnodeHandler {
  handleRequest(method: string, params: any[]): Promise<string>
}

/**
 * Interface defining all Web 3 methods a handler must support.
 */
export interface Web3Handler {
  blockNumber(): Promise<string>
  call(callObj: {}, defaultBlock: string): Promise<string>
  estimateGas(txObject: {}, defaultBlock: string): Promise<string>
  gasPrice(): Promise<string>
  getCode(address: Address, defaultBlock: string): Promise<string>
  getExecutionManagerAddress()
  getTransactionCount(address: Address, defaultBlock: string): Promise<string>
  getTransactionReceipt(txHash: string): Promise<string>
  networkVersion(): Promise<string>
  sendRawTransaction(signedTx: string): Promise<string>
}

// Enum of supported web3 rpc methods
export enum Web3RpcMethods {
  getTransactionCount = 'eth_getTransactionCount',
  sendRawTransaction = 'eth_sendRawTransaction',
  call = 'eth_call',
  getTransactionReceipt = 'eth_getTransactionReceipt',
  blockNumber = 'eth_blockNumber',
  gasPrice = 'eth_gasPrice',
  estimateGas = 'eth_estimateGas',
  getCode = 'eth_getCode',
  getExecutionManagerAddress = 'ovm_getExecutionManagerAddress',
  networkVersion = 'net_version',
}
