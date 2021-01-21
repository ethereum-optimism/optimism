/**
 * Copyright 2020, Optimism PBC
 * MIT License
 * https://github.com/ethereum-optimism
 */

import { remove0x } from '@eth-optimism/core-utils'
import { Bytes } from '@ethersproject/bytes'
import { AbiCoder } from '@ethersproject/abi'
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
  from: true,
}

export function serializeEthSignTransaction(transaction): Bytes {
  const abi = new AbiCoder()
  const encoded = abi.encode(
    ['uint256', 'uint256', 'uint256', 'uint256', 'address', 'bytes'],
    [
      transaction.nonce,
      transaction.gasLimit,
      transaction.gasPrice,
      transaction.chainId,
      transaction.to,
      transaction.data,
    ]
  )

  return Buffer.from(encoded.slice(2), 'hex')
}

// Use this function as input to `eth_sign`. It does not
// add the prefix because `eth_sign` does that. It does
// serialize the transaction and hash the serialized
// transaction.
export function sighashEthSign(transaction): Buffer {
  const serialized = serializeEthSignTransaction(transaction)
  const hash = remove0x(keccak256(serialized))
  return Buffer.from(hash, 'hex')
}
