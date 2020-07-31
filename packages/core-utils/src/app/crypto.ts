/* External Imports */
import { Md5 } from 'ts-md5'
import { ethers } from 'ethers'
import { TransactionRequest } from 'ethers/providers/abstract-provider'
import {
  joinSignature,
  resolveProperties,
  serializeTransaction,
} from 'ethers/utils'

/* Internal Imports */
import { HashAlgorithm, HashFunction } from '../types'
import { add0x, numberToHexString, remove0x } from './misc'

/**
 * Creates an Md5 hash of the provided input
 *
 * @param preimage the Buffer to hash
 * @returns the hash as a Buffer
 */
export const Md5Hash = (preimage: string): string => {
  return Md5.hashStr(preimage).toString()
}

/**
 * Computes the keccak256 hash of a value.
 * Note: assumes the value is a valid hex string.
 *
 * @param value Value to hash
 * @returns the hash of the value.
 */
export const keccak256 = (value: string): string => {
  const preimage = add0x(value)
  return remove0x(ethers.utils.keccak256(preimage))
}

/**
 * Computes the keccak256 hash of the value represented by the provided UTF-8 string.
 *
 * @param s The UTF-8 encoded string.
 * @returns The keccak256 hash.
 */
export const keccak256FromUtf8 = (s: string): string => {
  return add0x(keccak256(Buffer.from(s).toString('hex')))
}

/**
 * Gets the hash function for the provided HashAlgorithm.
 *
 * @param algo The HashAlgorithm in question
 * @returns The hash function, if one exists
 */
export const hashFunctionFor = (algo: HashAlgorithm): HashFunction => {
  switch (algo) {
    case HashAlgorithm.MD5:
      return Md5Hash
    case HashAlgorithm.KECCAK256:
      return keccak256
    default:
      throw Error(`HashAlgorithm ${algo} not supported.`)
  }
}

/**
 * Gets the tx signer address from the Tx Request and r, s, v.
 *
 * @param tx The Transaction Request.
 * @param r The r parameter of the signature.
 * @param s The s parameter of the signature.
 * @param v The v parameter of the signature.
 * @returns The signer's address.
 */
export const getTxSigner = async (
  tx: TransactionRequest,
  r: string,
  s: string,
  v: number
): Promise<string> => {
  const txHash: string = ethers.utils.keccak256(
    serializeTransaction(await resolveProperties(tx))
  )

  try {
    return ethers.utils.recoverAddress(
      ethers.utils.arrayify(txHash),
      joinSignature({
        s: add0x(s),
        r: add0x(r),
        v,
      })
    )
  } catch (e) {
    return undefined
  }
}

/**
 * Turns a signature's R, S, V values into the concatenated 65-byte signature.
 *
 * @param r The R value.
 * @param s The S value.
 * @param v The V value.
 * @returns the signature.
 * @throws Error If r, s, or v are not the correct length.
 */
export const rsvToSignature = (r: string, s: string, v: number): string => {
  const vString = remove0x(numberToHexString(v, 1))
  const sig = `${add0x(r)}${remove0x(s)}${vString}`
  // '0x' + 64 chars for r + 64 chars for s + 2 chars for v = 132
  if (sig.length !== 132) {
    throw Error(
      `Invalid v [${vString}], r [${r}], s[${s}]. v [${vString.length}] should be 2 chars, r [${r.length}] should be 64 chars, and s [${s.length}] should be 64 chars.`
    )
  }

  return sig
}

/**
 * Parses a signature in R || S || V format to R, S, and V.
 *
 * @param sig The signature to parse.
 * @returns an object with R, S, and V.
 * @throws Error if signature is not the correct length.
 */
export const signatureToRSV = (
  sig: string
): { r: string; s: string; v: number } => {
  const unprefixed: string = remove0x(sig)
  if (unprefixed.length !== 130) {
    throw Error(
      `Invalid signature. Must be 132 chars if prefixed with 0x and 130 otherwise. Got ${sig.length} chars. Signature: ${sig}`
    )
  }

  return {
    r: unprefixed.substr(0, 64),
    s: unprefixed.substr(64, 64),
    v: parseInt(unprefixed.substr(128, 2), 16),
  }
}
