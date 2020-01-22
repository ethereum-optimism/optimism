/* External Imports */
import { Address } from '@pigi/rollup-core/'
import {
  getLogger,
  add0x,
  abi,
  keccak256,
  remove0x,
  hexStrToBuf,
  bufToHexString,
  bufferUtils,
} from '@pigi/core-utils'
import * as ethereumjsAbi from 'ethereumjs-abi'
import { Contract, ContractFactory, Wallet } from 'ethers'
import { Provider } from 'ethers/providers'

/* Contract Imports */
import * as SimpleStorage from '../build/contracts/SimpleStorage.json'

const log = getLogger('helpers', true)

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export const manuallyDeployOvmContract = async (
  wallet: Wallet,
  provider: Provider,
  executionManager: Contract,
  contractDefinition,
  constructorArguments: any[]
): Promise<Address> => {
  const initcode = new ContractFactory(
    contractDefinition.abi,
    contractDefinition.bytecode
  ).getDeployTransaction(...constructorArguments).data as string

  const executeCallMethodId: string = ethereumjsAbi
    .methodID('executeCall', [])
    .toString('hex')

  const ovmCreateMethodId: string = ethereumjsAbi
    .methodID('ovmCREATE', [])
    .toString('hex')

  const timestamp: string = '00'.repeat(32)
  const origin: string = '00'.repeat(32)

  const emAddress: string = '00'.repeat(32)

  const data = `0x${executeCallMethodId}${timestamp}${origin}${emAddress}${remove0x(
    initcode
  )}`

  // Now actually apply it to our execution manager
  const tx = await wallet.sendTransaction({
    to: executionManager.address,
    data,
    gasLimit: 6_700_000,
  })

  // Extract the resulting ovm contract address
  const receipt = await provider.getTransactionReceipt(tx.hash)
  const createContractEventTypes = ['address', 'address', 'bytes32']
  const ovmContractAddress: Address = abi.decode(
    createContractEventTypes,
    receipt.logs[0].data
  )[0] // The OVM address is the first one in the list

  log.info('Deployed new contract at OVM address:', ovmContractAddress)
  return ovmContractAddress
}

/**
 * Creates an unsigned transaction.
 * @param {ethers.Contract} contract
 * @param {String} functionName
 * @param {Array} args
 */
export const getUnsignedTransactionCalldata = (
  contract,
  functionName,
  args
) => {
  return contract.interface.functions[functionName].encode(args)
}

/**
 * Deterministically computes the smart contract address given
 * the account that will deploy the contract (factory contract)
 * the salt as uint256 and the contract bytecode
 * Source: https://github.com/miguelmota/solidity-create2-example
 * Note: Use this function to generate new tests
 */
export const buildCreate2Address = (
  creatorAddress,
  saltHex,
  byteCode
): Address => {
  const preimage: string = `ff${remove0x(creatorAddress)}${remove0x(
    saltHex
  )}${keccak256(byteCode)}`
  return add0x(
    keccak256(preimage)
      .slice(-40)
      .toLowerCase()
  )
}

/**
 * Gets an address string from a bytes32 big-endian Address.
 * @param bytes32Address The 32-byte address string
 * @returns The 0x-prefixed 20-byte address string
 */
export const bytes32AddressToAddress = (bytes32Address: string): Address => {
  return bufToHexString(hexStrToBuf(bytes32Address).slice(12)).toLowerCase()
}

/**
 * Gets a padded big-endian 32-byte address string from an address string.
 * @param addr The 20-byte address string
 * @returns The 0x-prefixed 32-byte address string
 */
export const addressTobytes32Address = (addr: Address): string => {
  return bufToHexString(
    bufferUtils.padLeft(hexStrToBuf(addr), 32)
  ).toLowerCase()
}
