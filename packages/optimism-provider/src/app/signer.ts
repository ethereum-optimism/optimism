/**
 * Optimism Copyright 2020
 * MIT License
 */

import * as bio from '@bitrelay/bufio'
import {
  JsonRpcSigner,
  JsonRpcProvider,
  Web3Provider,
} from '@ethersproject/providers'
import { Logger } from '@ethersproject/logger'
import {
  BlockTag,
  Provider,
  TransactionRequest,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { BigNumberish, BigNumber } from '@ethersproject/bignumber'
import { Bytes, splitSignature } from '@ethersproject/bytes'
import { serialize, UnsignedTransaction } from '@ethersproject/transactions'
import { hexStrToBuf, isHexString, remove0x } from '@eth-optimism/core-utils'
import { ConnectionInfo, fetchJson, poll } from '@ethersproject/web'
import { keccak256 } from '@ethersproject/keccak256'

import {
  checkProperties,
  deepCopy,
  Deferrable,
  defineReadOnly,
  getStatic,
  resolveProperties,
  shallowCopy,
} from '@ethersproject/properties'

import {
  allowedTransactionKeys,
  serializeEthSignTransaction,
  sighashEthSign,
} from './utils'

import { OptimismProvider } from './provider'
import pkg = require('../../package.json')

const version = pkg.version
const logger = new Logger(version)

/**
 * OptimismSigner must be passed a Web3Provider that is responsible for key
 * management. Calls such as `eth_sendTransaction` must be sent to an optimism
 * node.
 */
export class OptimismSigner implements JsonRpcSigner {
  private _signer: JsonRpcSigner
  public readonly provider: Web3Provider
  private readonly _optimism: OptimismProvider

  public readonly _isSigner: boolean
  public readonly _index: number
  public readonly _address: string

  constructor(
    provider: Web3Provider,
    optimism: OptimismProvider,
    addressOrIndex: string | number
  ) {
    if (addressOrIndex == null) {
      addressOrIndex = 0
    }

    if (typeof addressOrIndex === 'string') {
      this._address = this.provider.formatter.address(addressOrIndex)
      this._index = null
    } else if (typeof addressOrIndex === 'number') {
      this._index = addressOrIndex
      this._address = null
    } else {
      logger.throwArgumentError(
        'invalid address or index',
        'addressOrIndex',
        addressOrIndex
      )
    }

    this._isSigner = true
    this._optimism = optimism
    this._signer = provider.getSigner()
    this.provider = provider
  }

  get signer() {
    return this._signer
  }

  get optimism() {
    return this._optimism
  }

  public connect(provider: Provider): JsonRpcSigner {
    return this.signer.connect(provider)
  }

  public connectUnchecked() {
    return this.signer.connectUnchecked()
  }

  public async getAddress(): Promise<string> {
    return this.signer.getAddress()
  }

  // This codepath expects an unsigned transaction.
  // TODO(mark): I think this codepath requires `eth_sendRawEthSignTransaction`
  public async sendUncheckedTransaction(
    transaction: Deferrable<TransactionRequest>
  ): Promise<string> {
    transaction = shallowCopy(transaction)

    let fromAddress = await this.getAddress()
    if (fromAddress) {
      fromAddress = fromAddress.toLowerCase()
    }

    // The JSON-RPC for eth_sendTransaction uses 90000 gas; if the user
    // wishes to use this, it is easy to specify explicitly, otherwise
    // we look it up for them.
    if (transaction.gasLimit == null) {
      const estimate = shallowCopy(transaction)
      estimate.from = fromAddress
      transaction.gasLimit = this.optimism.estimateGas(estimate)
    }

    // TODO(mark): Refactor this after tests
    return resolveProperties({
      tx: resolveProperties(transaction),
      sender: fromAddress,
    }).then(({ tx, sender }) => {
      if (tx.from != null) {
        if (tx.from.toLowerCase() !== sender) {
          logger.throwArgumentError(
            'from address mismatch',
            'transaction',
            transaction
          )
        }
      } else {
        tx.from = sender
      }

      const hexTx = (this.optimism.constructor as any).hexlifyTransaction(tx, {
        from: true,
      })

      // TODO: replace this
      return this.optimism.send('eth_sendTransaction', [hexTx]).then(
        (hash) => {
          return hash
        },
        (error) => {
          if (error.responseText) {
            // See: JsonRpcProvider.sendTransaction (@TODO: Expose a ._throwError??)
            if (error.responseText.indexOf('insufficient funds') >= 0) {
              logger.throwError(
                'insufficient funds',
                Logger.errors.INSUFFICIENT_FUNDS,
                {
                  transaction: tx,
                }
              )
            }
            if (error.responseText.indexOf('nonce too low') >= 0) {
              logger.throwError(
                'nonce has already been used',
                Logger.errors.NONCE_EXPIRED,
                {
                  transaction: tx,
                }
              )
            }
            if (
              error.responseText.indexOf(
                'replacement transaction underpriced'
              ) >= 0
            ) {
              logger.throwError(
                'replacement fee too low',
                Logger.errors.REPLACEMENT_UNDERPRICED,
                {
                  transaction: tx,
                }
              )
            }
          }
          throw error
        }
      )
    })
  }

  // Calls `eth_sign` on the web3 provider
  public async signTransaction(
    transaction: Deferrable<TransactionRequest>
  ): Promise<string> {
    const hash = sighashEthSign(transaction)
    const sig = await this.signer.signMessage(hash)

    if (transaction.chainId == null) {
      transaction.chainId = await this.getChainId()
    }

    // Copy over "allowed" properties into new object so that
    // `serialize` doesn't throw an error. A "from" property
    // may exist depending on the upstream codepath.
    const tx = {
      chainId: transaction.chainId,
      data: transaction.data,
      gasLimit: transaction.gasLimit,
      gasPrice: transaction.gasPrice,
      nonce: transaction.nonce,
      to: transaction.to,
      value: transaction.value,
    }

    return serialize(tx as UnsignedTransaction, sig)
  }

  // Populates all fields in a transaction, signs it and sends it to the network
  public async sendTransaction(
    transaction: Deferrable<TransactionRequest>
  ): Promise<TransactionResponse> {
    this._checkProvider('sendTransaction')
    const tx = await this.populateTransaction(transaction)
    const signed = await this.signTransaction(tx)
    return this.optimism.sendTransaction(signed)
  }

  public async signMessage(message: Bytes | string): Promise<string> {
    return this.signer.signMessage(message)
  }

  public async unlock(password: string): Promise<boolean> {
    return this.signer.unlock(password)
  }

  public _checkProvider(operation?: string): void {
    if (!this.provider) {
      logger.throwError(
        'missing provider',
        Logger.errors.UNSUPPORTED_OPERATION,
        {
          operation: operation || '_checkProvider',
        }
      )
    }
  }

  public _checkOptimism(operation?: string): void {
    if (!this.optimism) {
      logger.throwError(
        'missing optimism provider',
        Logger.errors.UNSUPPORTED_OPERATION,
        {
          operation: operation || '_checkProvider',
        }
      )
    }
  }

  public static isSigner(value: any): value is Signer {
    return !!(value && value._isSigner)
  }

  // Calls the optimism node to check the signer's address balance
  public async getBalance(blockTag?: BlockTag): Promise<BigNumber> {
    this._checkOptimism('getBalance')
    return this.optimism.getBalance(this.getAddress(), blockTag)
  }

  // Calls the optimism node to check the signer's address transaction count
  public async getTransactionCount(blockTag?: BlockTag): Promise<number> {
    this._checkOptimism('getTransactionCount')
    return this.optimism.getTransactionCount(this.getAddress(), blockTag)
  }

  // Calls the optmism node to estimate a transaction's gas
  public async estimateGas(
    transaction: Deferrable<TransactionRequest>
  ): Promise<BigNumber> {
    this._checkOptimism('estimateGas')
    const tx = await resolveProperties(this.checkTransaction(transaction))
    return this.optimism.estimateGas(tx)
  }

  // Populates "from" if unspecified, and calls with the transation
  public async call(
    transaction: Deferrable<TransactionRequest>,
    blockTag?: BlockTag
  ): Promise<string> {
    this._checkProvider('call')
    const tx = await resolveProperties(this.checkTransaction(transaction))
    return this.optimism.call(tx, blockTag)
  }

  // Calls the optimism node to get the chainid
  public async getChainId(): Promise<number> {
    this._checkOptimism('getChainId')
    const network = await this.optimism.getNetwork()
    return network.chainId
  }

  // Calls the optimism node to get the gas price
  public async getGasPrice(): Promise<BigNumber> {
    this._checkOptimism('getGasPrice')
    return this.optimism.getGasPrice()
  }

  // Resolve ENS on the optimism node, if it exists
  public async resolveName(name: string): Promise<string> {
    this._checkOptimism('resolveName')
    return this.optimism.resolveName(name)
  }

  // Checks a transaction does not contain invalid keys and if
  // no "from" is provided, populates it.
  // - does NOT require a provider
  // - adds "from" is not present
  // - returns a COPY (safe to mutate the result)
  // By default called from: (overriding these prevents it)
  //   - call
  //   - estimateGas
  //   - populateTransaction (and therefor sendTransaction)
  public checkTransaction(
    transaction: Deferrable<TransactionRequest>
  ): Deferrable<TransactionRequest> {
    for (const key in transaction) {
      if (!(key in allowedTransactionKeys)) {
        logger.throwArgumentError(
          'invalid transaction key: ' + key,
          'transaction',
          transaction
        )
      }
    }

    const tx = shallowCopy(transaction)

    if (tx.from == null) {
      tx.from = this.getAddress()
    } else {
      // Make sure any provided address matches this signer
      tx.from = Promise.all([Promise.resolve(tx.from), this.getAddress()]).then(
        (result) => {
          if (result[0] !== result[1]) {
            logger.throwArgumentError(
              'from address mismatch',
              'transaction',
              transaction
            )
          }
          return result[0]
        }
      )
    }

    return tx
  }

  // Populates ALL keys for a transaction and checks that "from" matches
  // this Signer. Should be used by sendTransaction but NOT by signTransaction.
  // By default called from: (overriding these prevents it)
  //   - sendTransaction
  public async populateTransaction(
    transaction: Deferrable<TransactionRequest>
  ): Promise<TransactionRequest> {
    const tx: Deferrable<TransactionRequest> = await resolveProperties(
      this.checkTransaction(transaction)
    )

    if (tx.to != null) {
      tx.to = Promise.resolve(tx.to).then((to) => this.resolveName(to))
    }
    if (tx.gasPrice == null) {
      tx.gasPrice = this.getGasPrice()
    }
    if (tx.nonce == null) {
      tx.nonce = this.getTransactionCount('pending')
    }

    if (tx.gasLimit == null) {
      tx.gasLimit = this.estimateGas(tx).catch((error) => {
        return logger.throwError(
          'cannot estimate gas; transaction may fail or may require manual gas limit',
          Logger.errors.UNPREDICTABLE_GAS_LIMIT,
          {
            error,
            tx,
          }
        )
      })
    }

    if (tx.chainId == null) {
      tx.chainId = this.getChainId()
    } else {
      tx.chainId = Promise.all([
        Promise.resolve(tx.chainId),
        this.getChainId(),
      ]).then((results) => {
        if (results[1] !== 0 && results[0] !== results[1]) {
          logger.throwArgumentError(
            'chainId address mismatch',
            'transaction',
            transaction
          )
        }
        return results[0]
      })
    }

    return resolveProperties(tx)
  }
}
