/**
 * Optimism Copyright 2020
 * MIT License
 */

// TODO: delete dead code
import { JsonRpcSigner, JsonRpcProvider } from '@ethersproject/providers'
import { Logger } from "@ethersproject/logger";
import { BlockTag, Provider, TransactionRequest, TransactionResponse } from '@ethersproject/abstract-provider';
import { Signer } from '@ethersproject/abstract-signer';
import { BigNumberish, BigNumber } from "@ethersproject/bignumber";
import { arrayify, Bytes } from '@ethersproject/bytes'
import { hexStrToBuf } from '@eth-optimism/core-utils'
import { ConnectionInfo, fetchJson, poll } from "@ethersproject/web";
import { checkProperties, deepCopy, Deferrable, defineReadOnly, getStatic, resolveProperties, shallowCopy } from "@ethersproject/properties";
import * as bio from '@bitrelay/bufio'

import { OptimismProvider } from './provider'
import pkg = require('../../package.json')

const version = pkg.version
const logger = new Logger(version);

const allowedTransactionKeys: { [ key: string ]: boolean } = {
    chainId: true, data: true, gasLimit: true, gasPrice:true, nonce: true, to: true, value: true
}

export class OptimismSigner implements JsonRpcSigner {
  private _signer: JsonRpcSigner
  public readonly provider: JsonRpcProvider

  public _isSigner: boolean
  public _index: number
  public _address: string

  // TODO: this shouldn't be the optimism provider
  constructor(provider: OptimismProvider, signer: JsonRpcSigner, addressOrIndex: string | number) {
    if (addressOrIndex == null) { addressOrIndex = 0; }

    if (typeof(addressOrIndex) === "string") {
      defineReadOnly(this, "_address", this.provider.formatter.address(addressOrIndex));
      defineReadOnly(this, "_index", null);
    } else if (typeof(addressOrIndex) === "number") {
      defineReadOnly(this, "_index", addressOrIndex);
      defineReadOnly(this, "_address", null);
    } else {
      logger.throwArgumentError("invalid address or index", "addressOrIndex", addressOrIndex);
    }

    defineReadOnly(this, "_isSigner", true);

    this._isSigner = true
    this._signer = signer
  }

  get signer() {
    return this._signer
  }

  // TODO: I think this is the right codepath.
  // Connect to metamask
  public connect(provider: Provider): JsonRpcSigner {
    // Modify this so it can connect.
    return this.signer.connect(provider)
  }

  public connectUnchecked() {
    return this.signer.connectUnchecked()
  }

  public async getAddress(): Promise<string> {
    return this.signer.getAddress()
  }

  // TODO(mark): I think this codepath requires `eth_sendRawEthSignTransaction`
  public async sendUncheckedTransaction(transaction: Deferrable<TransactionRequest>): Promise<string> {
    transaction = shallowCopy(transaction);

    let fromAddress = await this.getAddress();
    if (fromAddress) {
      fromAddress = fromAddress.toLowerCase();
    }

    // The JSON-RPC for eth_sendTransaction uses 90000 gas; if the user
    // wishes to use this, it is easy to specify explicitly, otherwise
    // we look it up for them.
    if (transaction.gasLimit == null) {
      const estimate = shallowCopy(transaction);
      estimate.from = fromAddress;
      transaction.gasLimit = this.provider.estimateGas(estimate);
    }

    // TODO(mark): Refactor this after tests
    return resolveProperties({
      tx: resolveProperties(transaction),
      sender: fromAddress
    }).then(({ tx, sender }) => {
      if (tx.from != null) {
        if (tx.from.toLowerCase() !== sender) {
          logger.throwArgumentError("from address mismatch", "transaction", transaction);
        }
      } else {
        tx.from = sender;
      }

      const hexTx = (this.provider.constructor as any).hexlifyTransaction(tx, { from: true });

      return this.provider.send("eth_sendTransaction", [ hexTx ]).then((hash) => {
        return hash;
      }, (error) => {
        if (error.responseText) {
          // See: JsonRpcProvider.sendTransaction (@TODO: Expose a ._throwError??)
          if (error.responseText.indexOf("insufficient funds") >= 0) {
            logger.throwError("insufficient funds", Logger.errors.INSUFFICIENT_FUNDS, {
              transaction: tx
            });
          }
          if (error.responseText.indexOf("nonce too low") >= 0) {
            logger.throwError("nonce has already been used", Logger.errors.NONCE_EXPIRED, {
              transaction: tx
            });
          }
          if (error.responseText.indexOf("replacement transaction underpriced") >= 0) {
            logger.throwError("replacement fee too low", Logger.errors.REPLACEMENT_UNDERPRICED, {
              transaction: tx
            });
          }
        }
        throw error;
      });
    });
  }

  // Calls `eth_sign` on the web3 provider
  public signTransaction(transaction: Deferrable<TransactionRequest>): Promise<string> {
    transaction = ensureTransactionDefaults(transaction)

    const bw = bio.write();
    bw.writeU64(transaction.nonce as number)
    bw.writeBytes(toBuffer(transaction.gasPrice as BigNumberish))
    bw.writeBytes(toBuffer(transaction.gasLimit as BigNumberish))
    bw.writeBytes(hexStrToBuf(transaction.to as string))
    bw.writeBytes(toBuffer(transaction.value as BigNumberish))
    bw.writeBytes(transaction.data as Buffer)
    bw.writeU8(0)
    bw.writeU8(0)

    return this.signer.signMessage(bw.render())
  }

  // The transaction must be signed already
  public sendTransaction(transaction: Deferrable<TransactionRequest>): Promise<TransactionResponse> {

    // if not signed, sign it

    return this.sendUncheckedTransaction(transaction).then((hash) => {
      return poll(() => {
        return this.provider.getTransaction(hash).then((tx: TransactionResponse) => {
          if (tx === null) { return undefined; }
          return this.provider._wrapTransaction(tx, hash);
        });
      }, { onceBlock: this.provider }).catch((error: Error) => {
        (error as any).transactionHash = hash;
        throw error;
      });
    });
  }

  /*
  // TODO(mark): check this codepath
  // Populates all fields in a transaction, signs it and sends it to the network
  public async sendTransaction(transaction: Deferrable<TransactionRequest>): Promise<TransactionResponse> {
    this._checkProvider("sendTransaction");
    return this.populateTransaction(transaction).then((tx) => {
      return this.signTransaction(tx).then((signedTx) => {
        return this.provider.sendTransaction(signedTx);
      });
    });
  }
  */

  public async signMessage(message: Bytes | string): Promise<string> {
    return this.signer.signMessage(message)
  }

  public async unlock(password: string): Promise<boolean> {
    return this.signer.unlock(password)
  }

  public _checkProvider(operation?: string): void {
    if (!this.provider) { logger.throwError("missing provider", Logger.errors.UNSUPPORTED_OPERATION, {
      operation: (operation || "_checkProvider") });
    }
  }

  public static isSigner(value: any): value is Signer {
    return !!(value && value._isSigner);
  }

  // target: public node
  public async getBalance(blockTag?: BlockTag): Promise<BigNumber> {
    this._checkProvider("getBalance");
    return this.provider.getBalance(this.getAddress(), blockTag);
  }

  // target: public node
  public async getTransactionCount(blockTag?: BlockTag): Promise<number> {
    this._checkProvider("getTransactionCount");
    return this.provider.getTransactionCount(this.getAddress(), blockTag);
  }

  // TODO(mark): double check this method
  // Populates "from" if unspecified, and estimates the gas for the transation
  public async estimateGas(transaction: Deferrable<TransactionRequest>): Promise<BigNumber> {
    this._checkProvider("estimateGas");
    const tx = await resolveProperties(this.checkTransaction(transaction));
    return this.provider.estimateGas(tx);
  }

  // Populates "from" if unspecified, and calls with the transation
  public async call(transaction: Deferrable<TransactionRequest>, blockTag?: BlockTag): Promise<string> {
    this._checkProvider("call");
    const tx = await resolveProperties(this.checkTransaction(transaction));
    return this.provider.call(tx, blockTag);
  }

  public async getChainId(): Promise<number> {
    this._checkProvider("getChainId");
    const network = await this.provider.getNetwork();
    return network.chainId;
  }

  // target: public node
  public async getGasPrice(): Promise<BigNumber> {
    this._checkProvider("getGasPrice");
    return this.provider.getGasPrice();
  }

  // target: public node
  public async resolveName(name: string): Promise<string> {
    this._checkProvider("resolveName");
    return this.provider.resolveName(name);
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
  public checkTransaction(transaction: Deferrable<TransactionRequest>): Deferrable<TransactionRequest> {
    for (const key in transaction) {
      if (!(key in allowedTransactionKeys)) {
        logger.throwArgumentError("invalid transaction key: " + key, "transaction", transaction);
      }
    }

    const tx = shallowCopy(transaction);

    if (tx.from == null) {
      tx.from = this.getAddress();
    } else {
      // Make sure any provided address matches this signer
      tx.from = Promise.all([
        Promise.resolve(tx.from),
        this.getAddress()
      ]).then((result) => {
        if (result[0] !== result[1]) {
          logger.throwArgumentError("from address mismatch", "transaction", transaction);
        }
        return result[0];
      });
    }

    return tx;
  }

  // Populates ALL keys for a transaction and checks that "from" matches
  // this Signer. Should be used by sendTransaction but NOT by signTransaction.
  // By default called from: (overriding these prevents it)
  //   - sendTransaction
  public async populateTransaction(transaction: Deferrable<TransactionRequest>): Promise<TransactionRequest> {
    const tx: Deferrable<TransactionRequest> = await resolveProperties(this.checkTransaction(transaction))

    if (tx.to != null) { tx.to = Promise.resolve(tx.to).then((to) => this.resolveName(to)); }
    if (tx.gasPrice == null) { tx.gasPrice = this.getGasPrice(); }
    if (tx.nonce == null) { tx.nonce = this.getTransactionCount("pending"); }

    if (tx.gasLimit == null) {
      tx.gasLimit = this.estimateGas(tx).catch((error) => {
        return logger.throwError("cannot estimate gas; transaction may fail or may require manual gas limit", Logger.errors.UNPREDICTABLE_GAS_LIMIT, {
          error,
          tx
        });
      });
    }

    if (tx.chainId == null) {
      tx.chainId = this.getChainId();
    } else {
      tx.chainId = Promise.all([
        Promise.resolve(tx.chainId),
        this.getChainId()
      ]).then((results) => {
        if (results[1] !== 0 && results[0] !== results[1]) {
          logger.throwArgumentError("chainId address mismatch", "transaction", transaction);
        }
        return results[0];
      });
    }

    return resolveProperties(tx);
  }
}

// TODO(mark): this may be duplicate functionality to `this.checkTransaction`
function ensureTransactionDefaults(transaction: Deferrable<TransactionRequest>): Deferrable<TransactionRequest> {
  transaction = deepCopy(transaction);

  if (isNullorUndefined(transaction.to)) {
    transaction.to = '0x0000000000000000000000000000000000000000'
  }

  if (isNullorUndefined(transaction.nonce)) {
    transaction.nonce = 0
  }

  if (isNullorUndefined(transaction.gasLimit)) {
    transaction.gasLimit = 0
  }

  if (isNullorUndefined(transaction.gasPrice)) {
    transaction.gasPrice = 0
  }

  if (isNullorUndefined(transaction.data)) {
    transaction.data = Buffer.alloc(0)
  }

  if (isNullorUndefined(transaction.value)) {
    transaction.value = 0
  }

  if (isNullorUndefined(transaction.chainId)) {
    transaction.chainId = 1
  }

  return transaction
}

function isNullorUndefined(a: any): boolean {
  return a === null || a === undefined
}

function toBuffer(n: BigNumberish): Buffer {
  const bignum = BigNumber.from(n)
  const uint8array = arrayify(bignum)
  return Buffer.from(uint8array)
}
