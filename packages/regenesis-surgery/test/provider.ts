import path from 'path'

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
      input = require(path.resolve(dump))
    } else if (typeof dump === 'object') {
      input = dump
    }

    this.state = input.alloc ? input.alloc : input

    if (this.state === null) {
      throw new Error('Must initialize with genesis or state object')
    }

    this._isProvider = false
  }

  async getBalance(
    addressOrName: string,
    // eslint-disable-next-line
    blockTag?: number | string
  ): Promise<BigNumber> {
    addressOrName = addressOrName.toLowerCase()
    const address = remove0x(addressOrName)
    const account = this.state[address] || this.state[addressOrName]
    if (!account || account.balance === '') {
      return BigNumber.from(0)
    }
    return BigNumber.from(account.balance)
  }

  async getTransactionCount(
    addressOrName: string,
    // eslint-disable-next-line
    blockTag?: number | string
  ): Promise<number> {
    addressOrName = addressOrName.toLowerCase()
    const address = remove0x(addressOrName)
    const account = this.state[address] || this.state[addressOrName]
    if (!account) {
      return 0
    }
    if (typeof account.nonce === 'number') {
      return account.nonce
    }
    if (account.nonce === '') {
      return 0
    }
    if (typeof account.nonce === 'string') {
      return BigNumber.from(account.nonce).toNumber()
    }
    return 0
  }

  async getCode(addressOrName: string): Promise<string> {
    addressOrName = addressOrName.toLowerCase()
    const address = remove0x(addressOrName)
    const account = this.state[address] || this.state[addressOrName]
    if (!account) {
      return '0x'
    }
    if (typeof account.code === 'string') {
      return add0x(account.code)
    }
    return '0x'
  }

  async getStorageAt(
    addressOrName: string,
    position: BigNumber | number
  ): Promise<string> {
    addressOrName = addressOrName.toLowerCase()
    const address = remove0x(addressOrName)
    const account = this.state[address] || this.state[addressOrName]
    if (!account) {
      return '0x'
    }
    const bytes32 = bytes32ify(position)
    const storage =
      account.storage[remove0x(bytes32)] || account.storage[bytes32]
    if (!storage) {
      return '0x'
    }
    return add0x(storage)
  }

  async call(
    transaction: Deferrable<TransactionRequest>,
    blockTag?: BlockTag | Promise<BlockTag>
  ): Promise<string> {
    throw new Error(
      `Unsupported Method: call with args: transaction - ${transaction}, blockTag - ${blockTag}`
    )
  }

  async send(method: string, args: Array<any>): Promise<any> {
    switch (method) {
      case 'eth_getProof': {
        const address = args[0]
        if (!address) {
          throw new Error('Must pass address as first arg')
        }
        const account = this.state[remove0x(address)] || this.state[address]
        // The account doesn't exist or is an EOA
        if (!account || !account.code || account.code === '0x') {
          return {
            codeHash: add0x(KECCAK256_NULL_S),
            storageHash: add0x(KECCAK256_RLP_S),
          }
        }
        return {
          codeHash: ethers.utils.keccak256(add0x(account.code)),
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
    throw new Error(
      `Unsupported Method: sendTransaction with args: transaction - ${signedTransaction}`
    )
  }

  async estimateGas(): Promise<BigNumber> {
    return BigNumber.from(0)
  }

  async getBlock(
    blockHashOrBlockTag: BlockTag | string | Promise<BlockTag | string>
  ): Promise<Block> {
    throw new Error(
      `Unsupported Method: getBlock with args blockHashOrBlockTag - ${blockHashOrBlockTag}`
    )
  }
  async getBlockWithTransactions(
    blockHashOrBlockTag: BlockTag | string | Promise<BlockTag | string>
  ): Promise<BlockWithTransactions> {
    throw new Error(
      `Unsupported Method: getBlockWithTransactions with args blockHashOrBlockTag - ${blockHashOrBlockTag}`
    )
  }
  async getTransaction(transactionHash: string): Promise<TransactionResponse> {
    throw new Error(
      `Unsupported Method: getTransaction with args transactionHash - ${transactionHash}`
    )
  }
  async getTransactionReceipt(
    transactionHash: string
  ): Promise<TransactionReceipt> {
    throw new Error(
      `Unsupported Method: getTransactionReceipt with args transactionHash - ${transactionHash}`
    )
  }

  async getLogs(filter: Filter): Promise<Array<Log>> {
    throw new Error(`Unsupported Method: getLogs with args filter - ${filter}`)
  }

  async resolveName(name: string | Promise<string>): Promise<null | string> {
    throw new Error(`Unsupported Method: resolveName with args name - ${name}`)
  }
  async lookupAddress(
    address: string | Promise<string>
  ): Promise<null | string> {
    throw new Error(
      `Unsupported Method: lookupAddress with args address - ${address}`
    )
  }

  on(eventName: EventType, listener: Listener): Provider {
    throw new Error(
      `Unsupported Method: on  with args eventName - ${eventName}, listener - ${listener}`
    )
  }
  once(eventName: EventType, listener: Listener): Provider {
    throw new Error(
      `Unsupported Method: once with args eventName - ${eventName}, listener - ${listener}`
    )
  }
  emit(eventName: EventType, ...args: Array<any>): boolean {
    throw new Error(
      `Unsupported Method: emit  with args eventName - ${eventName}, args - ${args}`
    )
  }
  listenerCount(eventName?: EventType): number {
    throw new Error(
      `Unsupported Method: listenerCount with args eventName - ${eventName}`
    )
  }
  listeners(eventName?: EventType): Array<Listener> {
    throw new Error(
      `Unsupported Method: listeners  with args eventName - ${eventName}`
    )
  }
  off(eventName: EventType, listener?: Listener): Provider {
    throw new Error(
      `Unsupported Method: off with args eventName - ${eventName}, listener - ${listener}`
    )
  }
  removeAllListeners(eventName?: EventType): Provider {
    throw new Error(
      `Unsupported Method: removeAllListeners with args eventName - ${eventName}`
    )
  }
  addListener(eventName: EventType, listener: Listener): Provider {
    throw new Error(
      `Unsupported Method: addListener with args eventName - ${eventName}, listener - ${listener}`
    )
  }
  removeListener(eventName: EventType, listener: Listener): Provider {
    throw new Error(
      `Unsupported Method: removeListener with args eventName - ${eventName}, listener - ${listener}`
    )
  }

  async waitForTransaction(
    transactionHash: string,
    confirmations?: number,
    timeout?: number
  ): Promise<TransactionReceipt> {
    throw new Error(
      `Unsupported Method: waitForTransaction with args transactionHash - ${transactionHash}, confirmations - ${confirmations}, timeout - ${timeout}`
    )
  }

  readonly _isProvider: boolean
}
