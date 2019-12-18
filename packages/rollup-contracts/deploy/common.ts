// Make sure an environment argument was passed
import { config } from 'dotenv'
import { resolve } from 'path'
import { Contract, ContractFactory, ethers, Wallet } from 'ethers'
import { Provider } from 'ethers/providers'

/**
 * Makes sure the necessary environment parameters are defined and loads environment config.
 */
const checkParamsAndLoadConfig = () => {
  if (
    !process.argv.length ||
    process.argv[process.argv.length - 1].endsWith('.js')
  ) {
    console.log(
      '\n\nError: Environment argument not provided. Usage: "yarn run deploy:purity-checker <env>"\n'
    )
    process.exit(0)
  }

  // Get the environment and read the appropriate environment file
  const environment = process.argv[process.argv.length - 1]
  // Note: Path is from 'build/deploy/<script>.js'
  config({ path: resolve(__dirname, `../../config/.${environment}.env`) })
}

/**
 * Used by `deployContractsFunction` below to deploy a contract from a wallet and contract JSON.
 *
 * @param contractJson The json of the contract to deploy.
 * @param wallet The wallet used to deploy.
 * @param args Any necessary constructor args.
 * @returns the deployed Contract reference.
 */
export const deployContract = async (
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

/**
 * Handles deploying contracts by calling the provided `deployContractsFunction`.
 * This function loads all of the necessary config and context for a deployment,
 * allowing `deployContractsFunction` to focus on what is being deployed.
 * @param deployContractsFunction The function that dictates what is deployed
 */
export const deploy = async (
  deployContractsFunction: (w: Wallet) => Promise<void>
): Promise<void> => {
  // If this doesn't work, nothing will happen.
  checkParamsAndLoadConfig()

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
    deployContractsFunction(wallet)
  }, 5_000)
}
