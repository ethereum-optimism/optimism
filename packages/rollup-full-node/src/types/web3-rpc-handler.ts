// Web3 handler interface
import { Address } from '@eth-optimism/rollup-core'

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
  getBlockByNumber(defaultBlock: string, fullObjects: boolean): Promise<any>
  getCode(address: Address, defaultBlock: string): Promise<string>
  getExecutionManagerAddress()
  getLogs(filter: any): Promise<any[]>
  getTransactionCount(address: Address, defaultBlock: string): Promise<string>
  getTransactionReceipt(txHash: string): Promise<string>
  networkVersion(): Promise<string>
  sendRawTransaction(signedTx: string): Promise<string>
}

// Enum of supported web3 rpc methods
export enum Web3RpcMethods {
  blockNumber = 'eth_blockNumber',
  call = 'eth_call',
  estimateGas = 'eth_estimateGas',
  gasPrice = 'eth_gasPrice',
  getBlockByNumber = 'eth_getBlockByNumber',
  getCode = 'eth_getCode',
  getExecutionManagerAddress = 'ovm_getExecutionManagerAddress',
  getLogs = 'eth_getLogs',
  getTransactionCount = 'eth_getTransactionCount',
  getTransactionReceipt = 'eth_getTransactionReceipt',
  networkVersion = 'net_version',
  sendRawTransaction = 'eth_sendRawTransaction',

  // Test methods:
  snapshot = 'evm_snapshot',
  revert = 'evm_revert',
  mine = 'evm_mine',
  increaseTimestamp = 'evm_increaseTime',
}
