/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, utils} from 'ethers'
import chalk from 'chalk';

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'
import L1ERC20GatewayJson from '../artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'
import L2DepositedERC20Json from '../artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'

let Factory__L1ERC20: ContractFactory
let Factory__L2DepositedERC20: ContractFactory
let Factory__L1ERC20Gateway: ContractFactory

let L1ERC20: Contract
let L2DepositedERC20: Contract
let L1ERC20Gateway: Contract


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

      Factory__L2DepositedERC20 = new ContractFactory(
        L2DepositedERC20Json.abi,
        L2DepositedERC20Json.bytecode,
        (hre as any).deployConfig.deployer_l2
      )

      Factory__L1ERC20Gateway = new ContractFactory(
        L1ERC20GatewayJson.abi,
        L1ERC20GatewayJson.bytecode,
        (hre as any).deployConfig.deployer_l1
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
      // [l2MessengerAddress, name, symbol]
      L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
        (hre as any).deployConfig.l2MessengerAddress,
        tokenName,
        tokenSymbol,
        {gasLimit: 800000, gasPrice: 0}
      )
      await L2DepositedERC20.deployTransaction.wait()
      const L2DepositedERC20DeploymentSubmission: DeploymentSubmission = {
        ...L2DepositedERC20,
        receipt: L2DepositedERC20.receipt,
        address: L2DepositedERC20.address,
        abi: L2DepositedERC20Json.abi,
      };
      await hre.deployments.save('L2DepositedERC20', L2DepositedERC20DeploymentSubmission)
      console.log(`🌕 ${chalk.red('L2DepositedERC20 deployed to:')} ${chalk.green(L2DepositedERC20.address)}`)

      //Deploy a gateway for the new token
      // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
      L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
        L1ERC20.address,
        L2DepositedERC20.address,
        (hre as any).deployConfig.l1MessengerAddress
      )
      await L1ERC20Gateway.deployTransaction.wait()
      const L1ERC20GatewayDeploymentSubmission: DeploymentSubmission = {
        ...L1ERC20Gateway,
        receipt: L1ERC20Gateway.receipt,
        address: L1ERC20Gateway.address,
        abi: L1ERC20GatewayJson.abi,
      };
      await hre.deployments.save('L1ERC20Gateway', L1ERC20GatewayDeploymentSubmission)
      console.log(`🌕 ${chalk.red('L1ERC20Gateway deployed to:')} ${chalk.green(L1ERC20Gateway.address)}`)

      //Initialize the ERC20 for the new token
      const initL2ERC20TX = await L2DepositedERC20.init(
        L1ERC20Gateway.address,
        {gasLimit: 800000, gasPrice: 0}
      );
      await initL2ERC20TX.wait();
      console.log(`⭐️ ${chalk.blue('L2DepositedERC20 initialized tx hash:')} ${chalk.green(initL2ERC20TX.hash)}`)
    }
}

deployFn.tags = ['L1ERC20Gateway', 'L2DepositedERC20', 'L1ERC20', 'test']

export default deployFn
