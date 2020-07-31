/* Internal Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Signer } from 'ethers'
import { getContractDefinition } from '@eth-optimism/rollup-contracts'

/**
 * Generates a contract factory from a contract definiton.
 * @param definition Definition to generate a factory from.
 * @param signer Signer to attach to the factory.
 * @returns Contract factory for the definition.
 */
export const getContractFactoryFromDefinition = (
  definition: any,
  signer: Signer
): ContractFactory => {
  return new ethers.ContractFactory(
    definition.abi,
    definition.bytecode || definition.evm.bytecode.object,
    signer
  )
}

/**
 * Generates a contract factory from a contract name.
 * @param contract Contract name to generate a factory for.
 * @param signer Signer to attach to the factory.
 * @returns Contract factory for the given contract name.
 */
export const getContractFactory = (
  contract: string,
  signer: Signer
): ContractFactory => {
  const definition = getContractDefinition(contract)
  return getContractFactoryFromDefinition(definition, signer)
}
