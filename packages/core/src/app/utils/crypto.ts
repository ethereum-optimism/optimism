import { Md5 } from 'ts-md5'
import { HashAlgorithm, HashFunction } from '../../types/utils'
import { add0x, remove0x } from './misc'
import { ethers } from 'ethers'

/**
 * Checks that the message with the provided signature was signed by the private key
 * associated with the provided public key.
 *
 * @param signature the signed message
 * @param message the message in question
 * @param publicKey the public key to check the signature against
 * @returns true if the signature matches the message when decrypted by the publicKey
 */
export const verifySignature = (
  signature: any,
  message: any,
  publicKey: any
): boolean => {
  // TODO: Make this do actual signature checking
  return signature === message
}

/**
 * Signs the provided message with the provided key
 *
 * @param key the key with which the message should be signed
 * @param message the message to be signed
 *
 * @returns the signed message
 */
export const sign = (key: any, message: any): any => {
  // TODO: Actually sign
  return message
}

/**
 * Decrypts the provided encrypted message with the provided public key
 *
 * @param publickey the public key in question
 * @param encryptedMessage the encrypted message to decrypt
 */
export const decryptWithPublicKey = (
  publickey: any,
  encryptedMessage: any
): any => {
  // TODO: Actually decrypt
  return encryptedMessage
}

/**
 * Creates an Md5 hash of the provided input
 *
 * @param preimage the Buffer to hash
 * @returns the hash as a Buffer
 */
export const Md5Hash = (preimage: Buffer): Buffer => {
  return Buffer.from(Md5.hashStr(preimage.toString()) as string)
}

/**
 * Computes the keccak256 hash of a value.
 * @param value Value to hash
 * @returns the hash of the value.
 */
export const keccak256 = (value: Buffer): Buffer => {
  const preimage = add0x(value.toString('hex'))
  return Buffer.from(remove0x(ethers.utils.keccak256(preimage)), 'hex')
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
