/**
 * Optimism Copyright 2020
 * MIT License
 */
import { Networkish } from "@ethersproject/networks";
import * as bio from '@bitrelay/bufio'
import { hexStrToBuf, isHexString } from '@eth-optimism/core-utils'
import { arrayify, Bytes } from '@ethersproject/bytes'
import { BigNumberish, BigNumber } from "@ethersproject/bignumber";
import { Deferrable, deepCopy } from "@ethersproject/properties";
import { TransactionRequest } from '@ethersproject/abstract-provider';

const blacklist = new Set([
  'web3_sha3',
  'net_version',
  'net_peerCount',
  'net_listening',
  'eth_protocolVersion',
  'eth_syncing',
  'eth_mining',
  'eth_hashrate',
  'eth_accounts',
  'eth_getUncleCountByBlockHash',
  'eth_getUncleCountByBlockNumber',
  'eth_sign',
  'eth_signTransaction',
  'eth_getUncleByBlockHashAndIndex',
  'eth_getUncleByBlockNumberAndIndex',
  'eth_getCompilers',
  'eth_compileLLL',
  'eth_compileSolidity',
  'eth_compileSerpent',
  'eth_getWork',
  'eth_submitWork',
  'eth_submitHashrate',
  'db_putString',
  'db_getString',
  'db_putHex',
  'db_getHex',
  'shh_post',
  'shh_version',
  'shh_newIdentity',
  'shh_hasIdentity',
  'shh_newGroup',
  'shh_addToGroup',
  'shh_newFilter',
  'shh_uninstallFilter',
  'shh_getFilterChanges',
  'shh_getMessages',
]);

export function isBlacklistedMethod(method: string) {
  return blacklist.has(method)
}

export function isUrl(n: string): boolean {
  if (typeof n === 'string') {
    if (n.startsWith('http')) {
      return true
    }
  }

  return false
}

export const allowedTransactionKeys: { [ key: string ]: boolean } = {
    chainId: true,
    data: true,
    gasLimit: true,
    gasPrice:true,
    nonce: true,
    to: true,
    value: true
}

export function serializeEthSignTransaction(transaction): Bytes {
  const bw = bio.write();
  bw.writeU64(transaction.nonce as number)
  bw.writeBytes(toBuffer(transaction.gasPrice as BigNumberish))
  bw.writeBytes(toBuffer(transaction.gasLimit as BigNumberish))
  bw.writeBytes(hexStrToBuf(transaction.to as string))
  bw.writeBytes(toBuffer(transaction.value as BigNumberish))
  bw.writeBytes(toBuffer(transaction.data as Buffer))
  bw.writeU8(0)
  bw.writeU8(0)
  return bw.render()
}

function toBuffer(n: BigNumberish): Buffer {
  if (typeof n === 'string' && isHexString(n as string)) {
    return hexStrToBuf(n as string)
  }

  const bignum = BigNumber.from(n)
  const uint8array = arrayify(bignum)
  return Buffer.from(uint8array)
}


// TODO(mark): this may be duplicate functionality to `this.checkTransaction`
export function ensureTransactionDefaults(transaction: Deferrable<TransactionRequest>): Deferrable<TransactionRequest> {
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
