import { BigNumberish, BigNumber } from '@ethersproject/bignumber'
import { Interface } from '@ethersproject/abi'

const iface = new Interface([
  'function relayMessage(address,address,bytes,uint256)',
  'function relayMessage(uint256,address,address,uint256,uint256,bytes)',
])

const nonceMask = BigNumber.from(
  '0x0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
)

/**
 * Encodes the version into the nonce.
 *
 * @param nonce
 * @param version
 */
export const encodeVersionedNonce = (
  nonce: BigNumber | BigInt,
  version: BigNumber | BigInt
): BigNumber => {
  if (!BigNumber.isBigNumber(nonce)) {
    nonce = BigNumber.from(Number(nonce))
  }
  if (!BigNumber.isBigNumber(version)) {
    version = BigNumber.from(Number(version))
  }
  return version.or(nonce.shl(240))
}

export const encodeVersionedNonceBigInt = (
  nonce: BigNumber | BigInt,
  version: BigNumber | BigInt
): BigInt => {
  return BigInt(encodeVersionedNonce(nonce, version).toNumber())
}

/**
 * Decodes the version from the nonce and returns the unversioned nonce as well
 * as the version. The version is encoded in the first byte of
 * the nonce. Note that this nonce is the nonce held in the
 * CrossDomainMessenger.
 *
 * @param nonce
 */
export const decodeVersionedNonce = (
  nonce: BigNumber | BigInt
): {
  version: BigNumber
  nonce: BigNumber
} => {
  if (!BigNumber.isBigNumber(nonce)) {
    nonce = BigNumber.from(Number(nonce))
  }
  return {
    version: nonce.shr(240),
    nonce: nonce.and(nonceMask),
  }
}

export const decodeVersionedNonceBigInt = (
  nonce: BigNumber | BigInt
): {
  version: BigInt
  nonce: BigInt
} => {
  const data = decodeVersionedNonce(nonce)
  return {
    version: BigInt(data.version.toNumber()),
    nonce: BigInt(data.nonce.toNumber()),
  }
}

/**
 * Encodes a V1 cross domain message. This message format was used before
 * bedrock and does not support value transfer because ETH was represented as an
 * ERC20 natively.
 *
 * @param target    The target of the cross domain message
 * @param sender    The sender of the cross domain message
 * @param data      The data passed along with the cross domain message
 * @param nonce     The cross domain message nonce
 */
export const encodeCrossDomainMessageV0 = (
  target: string,
  sender: string,
  data: string,
  nonce: BigNumber | BigInt
) => {
  if (!BigNumber.isBigNumber(nonce)) {
    nonce = BigNumber.from(Number(nonce))
  }
  return iface.encodeFunctionData(
    'relayMessage(address,address,bytes,uint256)',
    [target, sender, data, nonce]
  )
}

/**
 * Encodes a V1 cross domain message. This message format shipped with bedrock
 * and supports value transfer with native ETH.
 *
 * @param nonce     The cross domain message nonce
 * @param sender    The sender of the cross domain message
 * @param target    The target of the cross domain message
 * @param value     The value being sent with the cross domain message
 * @param gasLimit  The gas limit of the cross domain execution
 * @param data      The data passed along with the cross domain message
 */
export const encodeCrossDomainMessageV1 = (
  nonce: BigNumber | BigInt,
  sender: string,
  target: string,
  value: BigNumberish | BigInt,
  gasLimit: BigNumberish | BigInt,
  data: string
) => {
  return iface.encodeFunctionData(
    'relayMessage(uint256,address,address,uint256,uint256,bytes)',
    [nonce, sender, target, value, gasLimit, data]
  )
}

/**
 * Encodes a cross domain message. The version byte in the nonce determines
 * the serialization format that is used.
 *
 * @param nonce     The cross domain message nonce
 * @param sender    The sender of the cross domain message
 * @param target    The target of the cross domain message
 * @param value     The value being sent with the cross domain message
 * @param gasLimit  The gas limit of the cross domain execution
 * @param data      The data passed along with the cross domain message
 */
export const encodeCrossDomainMessage = (
  nonce: BigNumber | BigInt,
  sender: string,
  target: string,
  value: BigNumber | BigInt,
  gasLimit: BigNumber | BigInt,
  data: string
) => {
  if (!BigNumber.isBigNumber(nonce)) {
    nonce = BigNumber.from(Number(nonce))
  }
  if (!BigNumber.isBigNumber(value)) {
    value = BigNumber.from(Number(value))
  }
  if (!BigNumber.isBigNumber(gasLimit)) {
    gasLimit = BigNumber.from(Number(gasLimit))
  }

  const { version } = decodeVersionedNonce(nonce)
  if (version.eq(0)) {
    return encodeCrossDomainMessageV0(target, sender, data, nonce)
  } else if (version.eq(1)) {
    return encodeCrossDomainMessageV1(
      nonce,
      sender,
      target,
      value,
      gasLimit,
      data
    )
  }
  throw new Error(`unknown version ${version.toString()}`)
}
