/* External Imports */
import { deploy, deployContract } from '@eth-optimism/core-utils'
import { Wallet } from 'ethers'

/* Internal Imports */
import { deployPurityChecker } from './purity-checker'
import * as ExecutionManager from '../build/contracts/ExecutionManager.json'
import { resolve } from 'path'
import { GAS_LIMIT, DEFAULT_OPCODE_WHITELIST_MASK } from '../src/app'

const executionManagerDeploymentFunction = async (
  wallet: Wallet
): Promise<string> => {
  console.log(`\nDeploying ExecutionManager!\n`)

  const purityCheckerContractAddress = await deployPurityChecker()

  const executionManager = await deployContract(
    ExecutionManager,
    wallet,
    DEFAULT_OPCODE_WHITELIST_MASK,
    purityCheckerContractAddress,
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
