/**
 * Optimism Copyright 2020
 * MIT License
 */
import { Networkish } from '@ethersproject/networks'
import * as bio from '@bitrelay/bufio'
import { hexStrToBuf, isHexString, remove0x } from '@eth-optimism/core-utils'
import { arrayify, Bytes, zeroPad } from '@ethersproject/bytes'
import { BigNumberish, BigNumber } from '@ethersproject/bignumber'
import { Deferrable, deepCopy } from '@ethersproject/properties'
import { TransactionRequest } from '@ethersproject/abstract-provider'
import { keccak256 } from '@ethersproject/keccak256'

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
])

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

export const allowedTransactionKeys: { [key: string]: boolean } = {
  chainId: true,
  data: true,
  gasLimit: true,
  gasPrice: true,
  nonce: true,
  to: true,
  value: true,
}

export function serializeEthSignTransaction(transaction): Bytes {
  const nonce = zeroPad(transaction.nonce, 32)
  const gasLimit = zeroPad(transaction.gasLimit, 32)
  const gasPrice = zeroPad(transaction.gasPrice, 32)
  const to = hexStrToBuf(transaction.to)
  const data = toBuffer(transaction.data)
  const chainId = zeroPad(transaction.chainId, 32)

  // 32 + 32 + 32 + 20 + 32
  const size = 148 + data.length
  const bw = bio.write(size)

  bw.writeBytes(Buffer.from(nonce))
  bw.writeBytes(Buffer.from(gasLimit))
  bw.writeBytes(Buffer.from(gasPrice))
  bw.writeBytes(to)
  bw.writeBytes(data)
  bw.writeBytes(Buffer.from(chainId))

  return bw.render()
}

// Use this function as input to `eth_sign`. It does not
// add the prefix because `eth_sign` does that. It does
// serialize the transaction and hash the serialized
// transaction.
export function sighashEthSign(transaction) {
    const serialized = serializeEthSignTransaction(transaction)
    return keccak256(serialized)
}

function toBuffer(n: BigNumberish): Buffer {
  if (typeof n === 'string' && isHexString(n as string)) {
    return hexStrToBuf(n as string)
  }

  const uint8array = arrayify(n)
  return Buffer.from(uint8array)
}
