import { ethers } from 'ethers'
import { BigNumber } from '@ethersproject/bignumber'
import { Deferrable } from '@ethersproject/properties'
import { Provider } from '@ethersproject/providers'
import {
  Provider as AbstractProvider,
  EventType,
  TransactionRequest,
  TransactionResponse,
  TransactionReceipt,
  Filter,
  Log,
  Block,
  BlockWithTransactions,
  BlockTag,
  Listener,
} from '@ethersproject/abstract-provider'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'

import { bytes32ify, remove0x, add0x } from '@eth-optimism/core-utils'

// Represents the ethereum state
export interface State {
  [address: string]: {
    nonce: number
    balance: string
    codeHash: string
    root: string
    code?: string
    storage?: {
      [key: string]: string
    }
  }
}

// Represents a genesis file that geth can consume
export interface Genesis {
  config: {
    chainId: number
    homesteadBlock: number
    eip150Block: number
    eip155Block: number
    eip158Block: number
    byzantiumBlock: number
    constantinopleBlock: number
    petersburgBlock: number
    istanbulBlock: number
    muirGlacierBlock: number
    clique: {
      period: number
      epoch: number
    }
  }
  difficulty: string
  gasLimit: string
  extraData: string
  alloc: State
}

export class GenesisJsonProvider implements AbstractProvider {
  state: State

  constructor(dump: string | Genesis | State) {
    let input
    if (typeof dump === 'string') {
      input = require(dump)
    } else if (typeof dump === 'object') {
      input = dump
    }

    this.state = input.alloc ? input.alloc : input

    if (this.state === null) {
      throw new Error('Must initialize with genesis or state object')
    }
  }

  async getBalance(
    addressOrName: string,
    blockTag?: BlockTag
  ): Promise<BigNumber> {
    const address = remove0x(addressOrName)
    const account = this.state[address]
    if (!account) {
      return BigNumber.from(0)
    }
    return BigNumber.from(account.balance)
  }

  async getTransactionCount(
    addressOrName: string,
    blockTag?: BlockTag
  ): Promise<number> {
    const address = remove0x(addressOrName)
    const account = this.state[address]
    if (!account) {
      return 0
    }
    return account.nonce
  }

  async getCode(addressOrName: string, blockTag?: BlockTag): Promise<string> {
    const address = remove0x(addressOrName)
    const account = this.state[address]
    if (!account) {
      return '0x'
    }
    return add0x(account.code)
  }

  async getStorageAt(
    addressOrName: string,
    position: BigNumber | number,
    blockTag?: BlockTag
  ): Promise<string> {
    const address = remove0x(addressOrName)
    const account = this.state[address]
    if (!account) {
      return '0x'
    }
    const bytes32 = bytes32ify(position)
    const storage = account.storage[remove0x(bytes32)]
    if (!storage) {
      return '0x'
    }
    return add0x(storage)
  }

  async call(
    transaction: Deferrable<TransactionRequest>,
    blockTag?: BlockTag | Promise<BlockTag>
  ): Promise<string> {
    throw new Error('Unsupported Method: call')
  }

  async send(method: string, args: Array<any>): Promise<any> {
    switch (method) {
      case 'eth_getProof': {
        const address = args[0]
        if (!address) {
          throw new Error('Must pass address as first arg')
        }
        const account = this.state[remove0x(address)]
        // The account doesn't exist or is an EOA
        if (!account || !account.code || account.code === '0x') {
          return {
            codeHash: add0x(KECCAK256_NULL_S),
            storageHash: add0x(KECCAK256_RLP_S),
          }
        }
        return {
          codeHash: ethers.utils.keccak256('0x' + account.code),
          storageHash: add0x(account.root),
        }
      }

      default:
        throw new Error(`Unsupported Method: send ${method}`)
    }
  }

  async getNetwork() {
    return undefined
  }

  async getBlockNumber(): Promise<number> {
    return 0
  }
  async getGasPrice(): Promise<BigNumber> {
    return BigNumber.from(0)
  }

  async getFeeData() {
    return undefined
  }

  async sendTransaction(
    signedTransaction: string | Promise<string>
  ): Promise<TransactionResponse> {
    throw new Error('Unsupported Method: sendTransaction')
  }

  async estimateGas(
    transaction: Deferrable<TransactionRequest>
  ): Promise<BigNumber> {
    return BigNumber.from(0)
  }

  async getBlock(
    blockHashOrBlockTag: BlockTag | string | Promise<BlockTag | string>
  ): Promise<Block> {
    throw new Error('Unsupported Method: getBlock')
  }
  async getBlockWithTransactions(
    blockHashOrBlockTag: BlockTag | string | Promise<BlockTag | string>
  ): Promise<BlockWithTransactions> {
    throw new Error('Unsupported Method: getBlockWithTransactions')
  }
  async getTransaction(transactionHash: string): Promise<TransactionResponse> {
    throw new Error('Unsupported Method: getTransaction')
  }
  async getTransactionReceipt(
    transactionHash: string
  ): Promise<TransactionReceipt> {
    throw new Error('Unsupported Method: getTransactionReceipt')
  }

  async getLogs(filter: Filter): Promise<Array<Log>> {
    throw new Error('Unsupported Method: getLogs')
  }

  async resolveName(name: string | Promise<string>): Promise<null | string> {
    throw new Error('Unsupported Method: resolveName')
  }
  async lookupAddress(
    address: string | Promise<string>
  ): Promise<null | string> {
    throw new Error('Unsupported Method: lookupAddress')
  }

  on(eventName: EventType, listener: Listener): Provider {
    throw new Error('Unsupported Method: on')
  }
  once(eventName: EventType, listener: Listener): Provider {
    throw new Error('Unsupported Method: once')
  }
  emit(eventName: EventType, ...args: Array<any>): boolean {
    throw new Error('Unsupported Method: emit')
  }
  listenerCount(eventName?: EventType): number {
    throw new Error('Unsupported Method: listenerCount')
  }
  listeners(eventName?: EventType): Array<Listener> {
    throw new Error('Unsupported Method: listeners')
  }
  off(eventName: EventType, listener?: Listener): Provider {
    throw new Error('Unsupported Method: off')
  }
  removeAllListeners(eventName?: EventType): Provider {
    throw new Error('Unsupported Method: removeAllListeners')
  }
  addListener(eventName: EventType, listener: Listener): Provider {
    throw new Error('Unsupported Method: addListener')
  }
  removeListener(eventName: EventType, listener: Listener): Provider {
    throw new Error('Unsupported Method: removeListener')
  }

  async waitForTransaction(
    transactionHash: string,
    confirmations?: number,
    timeout?: number
  ): Promise<TransactionReceipt> {
    throw new Error('Unsupported Method: waitForTransaction')
  }

  readonly _isProvider: boolean
}
