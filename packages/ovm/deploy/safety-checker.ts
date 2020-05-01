/* External Imports */
import { Address } from '@eth-optimism/rollup-core'
import { deploy, deployContract, add0x } from '@eth-optimism/core-utils'
import { Wallet } from 'ethers'

/* Internal Imports */
import * as SafetyChecker from '../build/contracts/SafetyChecker.json'
import { resolve } from 'path'

const safetyCheckerDeploymentFunction = async (
  wallet: Wallet
): Promise<Address> => {
  let safetyCheckerContractAddress =
    process.env.DEPLOY_SAFETY_CHECKER_CONTRACT_ADDRESS
  if (!safetyCheckerContractAddress) {
    console.log(`\nDeploying Safety Checker!\n`)

    // Default config whitelists all opcodes EXCEPT:
    //    ADDRESS, BALANCE, BLOCKHASH, CALLCODE, CALLER, COINBASE,
    //    CREATE, CREATE2, DELEGATECALL, DIFFICULTY, EXTCODECOPY, EXTCODESIZE,
    //    GASLIMIT, GASPRICE, NUMBER, ORIGIN, SELFDESTRUCT, SLOAD, SSTORE,
    //    STATICCALL, TIMESTAMP
    // See test/contracts/whitelist-mask-generator.spec.ts for more info
    const whitelistMask =
      process.env.OPCODE_WHITELIST_MASK ||
      '0x600a0000000000000000001fffffffffffffffff0fcf000063f000013fff0fff'

    const executionManagerAddress =
      process.env.EXECUTION_MANAGER_ADDRESS || add0x('12'.repeat(20))

    console.log(
      `Deploying Safety Checker using mask '${whitelistMask}' and execution manager '${executionManagerAddress}'...`
    )
    whitelistMask
    const safetyChecker = await deployContract(
      SafetyChecker,
      wallet,
      whitelistMask,
      executionManagerAddress
    )
    safetyCheckerContractAddress = safetyChecker.address

    console.log(
      `Safety Checker deployed to ${safetyCheckerContractAddress}!\n\n`
    )
  } else {
    console.log(
      `Using Safety Checker contract at ${safetyCheckerContractAddress}\n`
    )
  }
  return safetyCheckerContractAddress
}

/**
 * Deploys the Safety Checker contract.
 *
 * @param rootContract Whether or not this is the main contract being deployed (as compared to a dependency).
 * @returns The deployed contract's address.
 */
export const deploySafetyChecker = async (
  rootContract: boolean = false
): Promise<string> => {
  // Note: Path is from 'build/deploy/<script>.js'
  const configDirPath = resolve(__dirname, `../../config/`)

  return deploy(safetyCheckerDeploymentFunction, configDirPath, rootContract)
}

deploySafetyChecker(true)
