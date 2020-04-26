// Web3 handler interface
import { Address } from '@eth-optimism/rollup-core'

export interface FullnodeHandler {
  handleRequest(
    method: string,
    params: any[],
    requesterIpAddress?: string
  ): Promise<string>
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
  getBlockByHash(blockHash: string, fullObjects: boolean): Promise<any>
  getCode(address: Address, defaultBlock: string): Promise<string>
  getExecutionManagerAddress()
  getLogs(ovmFilter: any): Promise<any[]>
  getTransactionByHash(transactionHash: string): Promise<any>
  getTransactionCount(address: Address, defaultBlock: string): Promise<string>
  getTransactionReceipt(txHash: string): Promise<string>
  networkVersion(): Promise<string>
  sendRawTransaction(signedTx: string): Promise<string>
  chainId(): Promise<string>
}

// Enum of supported web3 rpc methods
export enum Web3RpcMethods {
  blockNumber = 'eth_blockNumber',
  call = 'eth_call',
  estimateGas = 'eth_estimateGas',
  gasPrice = 'eth_gasPrice',
  getBlockByNumber = 'eth_getBlockByNumber',
  getBlockByHash = 'eth_getBlockByHash',
  getBalance = 'eth_getBalance',
  getCode = 'eth_getCode',
  getExecutionManagerAddress = 'ovm_getExecutionManagerAddress',
  getLogs = 'eth_getLogs',
  getTransactionByHash = 'eth_getTransactionByHash',
  getTransactionCount = 'eth_getTransactionCount',
  getTransactionReceipt = 'eth_getTransactionReceipt',
  networkVersion = 'net_version',
  sendTransaction = 'eth_sendTransaction',
  sendRawTransaction = 'eth_sendRawTransaction',
  chainId = 'eth_chainId',

  // Test methods:
  accounts = 'eth_accounts',
  snapshot = 'evm_snapshot',
  revert = 'evm_revert',
  mine = 'evm_mine',
  increaseTimestamp = 'evm_increaseTime',
}

export const allWeb3RpcMethodsIncludingTest = Object.values(Web3RpcMethods)
export const testWeb3RpcMethods = Object.values([
  Web3RpcMethods.accounts,
  Web3RpcMethods.snapshot,
  Web3RpcMethods.revert,
  Web3RpcMethods.mine,
  Web3RpcMethods.increaseTimestamp,
])
export const web3RpcMethodsExcludingTest = allWeb3RpcMethodsIncludingTest.filter(
  (x) => testWeb3RpcMethods.indexOf(x) < 0
)
