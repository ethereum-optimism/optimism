import { Wallet } from 'ethers'
import { deploy, deployContract } from './common'

import * as PurityChecker from '../build/contracts/PurityChecker.json'

const deployContracts = async (wallet: Wallet): Promise<void> => {
  let purityCheckerContractAddress =
    process.env.DEPLOY_PURITY_CHECKER_CONTRACT_ADDRESS
  if (!purityCheckerContractAddress) {
    // Default config whitelists all opcodes EXCEPT:
    //    ADDRESS, BALANCE, BLOCKHASH, CALLCODE, CALLER, CALLVALUE, COINBASE,
    //    CREATE, CREATE2, DELEGATECALL, DIFFICULTY, EXTCODECOPY, EXTCODESIZE,
    //    GAS, GASLIMIT, GASPRICE, NUMBER, ORIGIN, SELFDESTRUCT, SLOAD, SSTORE,
    //    STATICCALL, TIMESTAMP
    // See test/purity-checker/whitelist-mask-generator.spec.ts for more info
    const whitelistMask =
      process.env.OPCODE_WHITELIST_MASK ||
      '0x600a0000000000000000001fffffffffffffffff0bcf000063e000013fff0fff'

    console.log(`Deploying OpcodeWhitelist using mask '${whitelistMask}'...`)

    const purityChecker = await deployContract(
      PurityChecker,
      wallet,
      whitelistMask
    )
    purityCheckerContractAddress = purityChecker.address

    console.log(
      `OpcodeWhitelist deployed to ${purityCheckerContractAddress}!\n\n`
    )
  } else {
    console.log(
      `Using OpcodeWhitelist contract at ${purityCheckerContractAddress}\n`
    )
  }

  // TODO: Deploy other stuff that depends on whitelist
}

deploy(deployContracts)
