import { ContractFactory } from 'ethers'
import { getLogger, add0x, abi } from '@pigi/core-utils'

const log = getLogger('helpers', true)

/**
 * Helper function for generating initcode based on a contract definition & constructor arguments
 */
export const manuallyDeployOvmContract = async (
  provider,
  executionManager,
  contractDefinition,
  constructorArguments
) => {
  const initcode = new ContractFactory(
    contractDefinition.abi,
    contractDefinition.bytecode
  ).getDeployTransaction(...constructorArguments).data
  const tx = await executionManager.executeTransaction(
    {
      ovmEntrypoint: add0x('00'.repeat(20)),
      ovmCalldata: initcode,
    },
    0,
    0
  )
  // Extract the resulting ovm contract address
  const reciept = await provider.getTransactionReceipt(tx.hash)
  const createContractEventTypes = ['address', 'address', 'bytes32']
  const ovmContractAddress = abi.decode(
    createContractEventTypes,
    reciept.logs[0].data
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
