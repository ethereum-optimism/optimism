import { Contract, ContractFactory, Wallet, ethers } from 'ethers'
import { config } from 'dotenv'
import { resolve } from 'path'

import * as RollupChain from '../build/RollupChain.json'
import * as UnipigTransitionEvaluator from '../build/UnipigTransitionEvaluator.json'
import * as RollupMerkleUtils from '../build/RollupMerkleUtils.json'
import { Provider } from 'ethers/providers'

// Make sure an environment argument was passed
if (
  !process.argv.length ||
  process.argv[process.argv.length - 1].endsWith('.js')
) {
  console.log(
    '\n\nError: Environment argument not provided. Usage: "yarn run deploy:rollup-chain <env>"\n'
  )
  process.exit(0)
}

// Get the environment and read the appropriate environment file
const environment = process.argv[process.argv.length - 1]
// Note: Path is from 'build/deploy/deploy-rollup-chain.js'
config({ path: resolve(__dirname, `../../config/.${environment}.env`) })

const deployContract = async (
  contractJson: any,
  wallet: Wallet,
  ...args: any
): Promise<Contract> => {
  const factory = new ContractFactory(
    contractJson.abi,
    contractJson.bytecode,
    wallet
  )
  const contract = await factory.deploy(...args)
  console.log(
    `Address: [${contract.address}], Tx: [${contract.deployTransaction.hash}]`
  )
  return contract.deployed()
}

const deployContracts = async (wallet: Wallet): Promise<void> => {
  let evaluatorContractAddress = process.env.DEPLOY_EVALUATOR_CONTRACT_ADDRESS
  if (!evaluatorContractAddress) {
    console.log('Deploying UnipigTransitionEvaluator...')
    const transitionEvaluator = await deployContract(
      UnipigTransitionEvaluator,
      wallet
    )
    evaluatorContractAddress = transitionEvaluator.address
    console.log('UnipigTransitionEvaluator deployed!\n\n')
  } else {
    console.log(
      `Using UnipigTransitionEvaluator contract at ${evaluatorContractAddress}\n`
    )
  }

  let merkleUtilsContractAddress =
    process.env.DEPLOY_MERKLE_UTILS_CONTRACT_ADDRESS
  if (!merkleUtilsContractAddress) {
    console.log('Deploying RollupMerkleUtils...')
    const merkleUtils = await deployContract(RollupMerkleUtils, wallet)
    merkleUtilsContractAddress = merkleUtils.address
    console.log('RollupMerkleUtils deployed!\n\n')
  } else {
    console.log(
      `Using RollupMerkleUtils contract at ${merkleUtilsContractAddress}\n`
    )
  }

  const aggregatorAddress: string = process.env.AGGREGATOR_ADDRESS

  console.log('Deploying RollupChain...')
  await deployContract(
    RollupChain,
    wallet,
    evaluatorContractAddress,
    merkleUtilsContractAddress,
    aggregatorAddress
  )
  console.log('RollupChain deployed!\n\n')
}

const deploy = async (): Promise<void> => {
  console.log(`\n\n********** STARTING DEPLOYMENT ***********\n\n`)
  // Make sure mnemonic exists
  const deployMnemonic = process.env.DEPLOY_MNEMONIC
  if (!deployMnemonic) {
    console.log(
      `Error: No DEPLOY_MNEMONIC env var set. Please add it to .<environment>.env file it and try again. See .env.example for more info.\n`
    )
    return
  }

  // Connect provider
  let provider: Provider
  const network = process.env.DEPLOY_NETWORK
  if (!network || network === 'local') {
    provider = new ethers.providers.JsonRpcProvider(
      process.env.DEPLOY_LOCAL_URL || 'http://127.0.0.1:7545'
    )
  } else {
    provider = ethers.getDefaultProvider(network)
  }

  // Create wallet
  const wallet = Wallet.fromMnemonic(deployMnemonic).connect(provider)

  console.log(`\nDeploying to network [${network || 'local'}] in 5 seconds!\n`)
  setTimeout(() => {
    deployContracts(wallet)
  }, 5_000)
}

deploy()
