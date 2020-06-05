/* External Imports */
import { deploy, deployContract } from '@eth-optimism/core-utils'
import { ExecutionManagerContractDefinition } from '@eth-optimism/rollup-contracts'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
} from '@eth-optimism/rollup-core'

import { Wallet } from 'ethers'
import { Provider } from 'ethers/providers'

/* Internal Imports */
import { deploySafetyChecker } from './safety-checker'
import { resolve } from 'path'

const executionManagerDeploymentFunction = async (
  wallet: Wallet,
  provider: Provider
): Promise<string> => {
  console.log(`\nDeploying ExecutionManager!\n`)

  const safetyCheckerContractAddress = await deploySafetyChecker()

  const executionManager = await deployContract(
    ExecutionManagerContractDefinition,
    wallet,
    DEFAULT_OPCODE_WHITELIST_MASK,
    safetyCheckerContractAddress,
    GAS_LIMIT,
    true
  )

  console.log(`Execution Manager deployed to ${executionManager.address}!\n\n`)

  return executionManager.address
}

/**
 * Deploys the ExecutionManager contract.
 *
 * @param rootContract Whether or not this is the main contract being deployed (as compared to a dependency).
 * @returns The deployed contract's address.
 */
export const deployExecutionManager = async (
  rootContract: boolean = false
): Promise<string> => {
  // Note: Path is from 'build/deploy/<script>.js'
  const configDirPath = resolve(__dirname, `../../config/`)

  return deploy(executionManagerDeploymentFunction, configDirPath, rootContract)
}

deployExecutionManager(true)
