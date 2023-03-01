import type { Address } from '@wagmi/core'
import { BigNumber } from 'ethers'

import { DataTypeOption } from '../types/DataTypeOption'
import { ParseBytesReturn } from '../types/ParseBytesReturn'
import { readAttestations } from './readAttestations'

/**
 * reads attestation from the attestation station contract
 *
 * @param attestationRead - the parameters for reading an attestation
 * @returns attestation result
 * @throws Error if key is longer than 32 bytes
 * @example
 * const attestation = await readAttestation(
 * {
 *  creator: creatorAddress,
 * about: aboutAddress,
 * key: 'my_key',
 * },
 */
export const readAttestation = async <TDataType extends DataTypeOption>(
  /**
   * Creator of the attestation
   */
  creator: Address,
  /**
   * Address the attestation is about
   */
  about: Address,
  /**
   * Key of the attestation
   */
  key: string,
  /**
   * Data type of the attestation
   * string | bool | number | address | bytes
   *
   * @defaults 'string'
   */
  dataType: TDataType,
  /**
   * Attestation address
   * defaults to the official Optimism attestation station determistic deploy address
   *
   * @defaults '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
   */
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
): Promise<ParseBytesReturn<TDataType>> => {
  const [result] = await readAttestations({
    creator,
    about,
    key,
    contractAddress,
    dataType,
  })
  return result as ParseBytesReturn<TDataType>
}

/**
 * Reads a string attestation
 */
export const readAttestationString = (
  /**
   * Creator of the attestation
   */
  creator: Address,
  /**
   * Address the attestation is about
   */
  about: Address,
  /**
   * Key of the attestation
   */
  key: string,
  /**
   * Attestation address
   * defaults to the official Optimism attestation station determistic deploy address
   *
   * @defaults '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
   */
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  return readAttestation(
    creator,
    about,
    key,
    'string',
    contractAddress
  ) as Promise<string>
}

export const readAttestationBool = (
  /**
   * Creator of the attestation
   */
  creator: Address,
  /**
   * Address the attestation is about
   */
  about: Address,
  /**
   * Key of the attestation
   */
  key: string,
  /**
   * Attestation address
   * defaults to the official Optimism attestation station determistic deploy address
   *
   * @defaults '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
   */
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  return readAttestation(
    /**
     * Creator of the attestation
     */
    creator,
    about,
    key,
    'bool',
    contractAddress
  ) as Promise<boolean>
}

export const readAttestationNumber = (
  /**
   * Creator of the attestation
   */
  creator: Address,
  /**
   * Address the attestation is about
   */
  about: Address,
  /**
   * Key of the attestation
   */
  key: string,
  /**
   * Attestation address
   * defaults to the official Optimism attestation station determistic deploy address
   *
   * @defaults '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
   */
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  return readAttestation(
    creator,
    about,
    key,
    'number',
    contractAddress
  ) as Promise<BigNumber>
}

export const readAttestationAddress = (
  /**
   * Creator of the attestation
   */
  creator: Address,
  /**
   * Address the attestation is about
   */
  about: Address,
  /**
   * Key of the attestation
   */
  key: string,
  /**
   * Attestation address
   * defaults to the official Optimism attestation station determistic deploy address
   *
   * @defaults '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
   */
  contractAddress: Address = '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'
) => {
  return readAttestation(
    creator,
    about,
    key,
    'address',
    contractAddress
  ) as Promise<Address>
}
