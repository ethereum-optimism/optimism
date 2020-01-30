/* External Imports */
import { Address } from '@pigi/rollup-core'
import { deploy, deployContract } from '@pigi/core-utils'
import { Wallet } from 'ethers'

/* Internal Imports */
import * as PurityChecker from '../build/contracts/PurityChecker.json'
import { resolve } from 'path'

const purityCheckerDeploymentFunction = async (
  wallet: Wallet
): Promise<Address> => {
  let purityCheckerContractAddress =
    process.env.DEPLOY_PURITY_CHECKER_CONTRACT_ADDRESS
  if (!purityCheckerContractAddress) {
    console.log(`\nDeploying Purity Checker!\n`)

    // Default config whitelists all opcodes EXCEPT:
    //    ADDRESS, BALANCE, BLOCKHASH, CALLCODE, CALLER, COINBASE,
    //    CREATE, CREATE2, DELEGATECALL, DIFFICULTY, GASLIMIT, GASPRICE,
    //    INVALID, NUMBER, ORIGIN, SELFDESTRUCT, SLOAD, SSTORE, STATICCALL, TIMESTAMP
    // See test/purity-checker/whitelist-mask-generator.spec.ts for more info
    const whitelistMask =
      process.env.OPCODE_WHITELIST_MASK ||
      '0x200a0000000000000000001fffffffffffffffff0fcf0000fbf000013fff0fff'

    console.log(`Deploying OpcodeWhitelist using mask '${whitelistMask}'...`)

    const purityChecker = await deployContract(
      PurityChecker,
      wallet,
      whitelistMask
    )
    purityCheckerContractAddress = purityChecker.address

    console.log(
      `Purity Checker deployed to ${purityCheckerContractAddress}!\n\n`
    )
  } else {
    console.log(
      `Using Purity Checker contract at ${purityCheckerContractAddress}\n`
    )
  }
  return purityCheckerContractAddress
}

/**
 * Deploys the Purity Checker contract.
 *
 * @param rootContract Whether or not this is the main contract being deployed (as compared to a dependency).
 * @returns The deployed contract's address.
 */
export const deployPurityChecker = async (
  rootContract: boolean = false
): Promise<string> => {
  // Note: Path is from 'build/deploy/<script>.js'
  const configDirPath = resolve(__dirname, `../../config/`)

  return deploy(purityCheckerDeploymentFunction, configDirPath, rootContract)
}

deployPurityChecker(true)
