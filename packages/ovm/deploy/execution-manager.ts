import { Wallet } from 'ethers'
import { deploy, deployContract } from './common'

import * as ExecutionManager from '../build/contracts/ExecutionManager.json'

const deployContracts = async (wallet: Wallet): Promise<void> => {
  const purityCheckerContractAddress =
    process.env.DEPLOY_PURITY_CHECKER_CONTRACT_ADDRESS

  const executionManager = await deployContract(
    ExecutionManager,
    wallet,
    purityCheckerContractAddress,
    wallet.address
  )

  console.log(`Execution Manager deployed to ${executionManager.address}!\n\n`)
}

deploy(deployContracts)
