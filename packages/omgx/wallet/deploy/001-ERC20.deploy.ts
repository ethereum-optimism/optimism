/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, utils} from 'ethers'
import chalk from 'chalk';
import { getContractFactory } from '@eth-optimism/contracts';

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'

let Factory__L1ERC20: ContractFactory
let Factory__L2ERC20: ContractFactory

let L1ERC20: Contract
let L2ERC20: Contract

//Test ERC20
const initialSupply = utils.parseEther("10000000000")
const tokenName = 'JLKN'
const tokenSymbol = 'JLKN'

const deployFn: DeployFunction = async (hre) => {
    // If TEST env var is not undefined we deploy these test contractgs
    if (process.env.TEST) {
      Factory__L1ERC20 = new ContractFactory(
        L1ERC20Json.abi,
        L1ERC20Json.bytecode,
        (hre as any).deployConfig.deployer_l1
      )

      Factory__L2ERC20 = getContractFactory(
        "L2StandardERC20",
        (hre as any).deployConfig.deployer_l2,
        true,
      )

      //Mint a new token on L1 and set up the L1 and L2 infrastructure
      // [initialSupply, name, symbol]
      // this is owned by bobl1Wallet
      L1ERC20 = await Factory__L1ERC20.deploy(
        initialSupply,
        tokenName,
        tokenSymbol
      )
      await L1ERC20.deployTransaction.wait()
      const L1ERC20DeploymentSubmission: DeploymentSubmission = {
        ...L1ERC20,
        receipt: L1ERC20.receipt,
        address: L1ERC20.address,
        abi: L1ERC20Json.abi,
      };
      await hre.deployments.save('L1ERC20', L1ERC20DeploymentSubmission)
      console.log(`🌕 ${chalk.red('L1ERC20 deployed to:')} ${chalk.green(L1ERC20.address)}`)

      //Set up things on L2 for this new token
      // [L2StandardBridgeAddress, L1TokenAddress, tokenName, tokenSymbol]
      L2ERC20 = await Factory__L2ERC20.deploy(
        (hre as any).deployConfig.L2StandardBridgeAddress,
        L1ERC20.address,
        tokenName,
        tokenSymbol,
        {gasLimit: 800000, gasPrice: 0}
      )
      await L2ERC20.deployTransaction.wait()
      const L2ERC20DeploymentSubmission: DeploymentSubmission = {
        ...L2ERC20,
        receipt: L2ERC20.receipt,
        address: L2ERC20.address,
        abi: L2ERC20.abi,
      };
      await hre.deployments.save('L2ERC20', L2ERC20DeploymentSubmission)
      console.log(`🌕 ${chalk.red('L2ERC20 deployed to:')} ${chalk.green(L2ERC20.address)}`)
    }
}

deployFn.tags = ['L1ERC20Gateway', 'L2DepositedERC20', 'L1ERC20', 'test']

export default deployFn
