/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, utils} from 'ethers'
import chalk from 'chalk';
import { getContractFactory } from '@eth-optimism/contracts';

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'
import preSupportedTokens from '../preSupportedTokens.json';

let Factory__L1ERC20: ContractFactory
let Factory__L2ERC20: ContractFactory

let L1ERC20: Contract
let L2ERC20: Contract

//Test ERC20
const initialSupply = utils.parseEther("10000000000")

const deployFn: DeployFunction = async (hre) => {

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

  let tokenAddress = null;

  for (let token of preSupportedTokens.supportedTokens) {
    
    if ((hre as any).deployConfig.network === 'local' || token.symbol === 'TEST') {
      //do not deploy existing tokens on Rinkeby or Mainnet
      //only deploy tokens if it's the TEST token or we are on local
      
      L1ERC20 = await Factory__L1ERC20.deploy(
        initialSupply,
        token.name,
        token.symbol,
      )
      await L1ERC20.deployTransaction.wait()

      tokenAddress = L1ERC20.address;

      const L1ERC20DeploymentSubmission: DeploymentSubmission = {
        ...L1ERC20,
        receipt: L1ERC20.receipt,
        address: L1ERC20.address,
        abi: L1ERC20Json.abi,
      };

      await hre.deployments.save(`TK_L1${token.symbol}`, L1ERC20DeploymentSubmission)
      console.log(`🌕 ${chalk.red(`L1 ${token.name} was newly deployed to`)} ${chalk.green(tokenAddress)}`)
    } else if ( (hre as any).deployConfig.network === 'rinkeby' ) {
      tokenAddress = token.address.rinkeby
      await hre.deployments.save(`TK_L1${token.symbol}`, { abi: L1ERC20Json.abi, address: tokenAddress })
      console.log(`🌕 ${chalk.red(`L1 ${token.name} is located at`)} ${chalk.green(tokenAddress)}`)
    } else if ( (hre as any).deployConfig.network === 'mainnet' ) {
      tokenAddress = token.address.mainnet
      await hre.deployments.save(`TK_L1${token.symbol}`, { abi: L1ERC20Json.abi, address: tokenAddress })
      console.log(`🌕 ${chalk.red(`L1 ${token.name} is located at`)} ${chalk.green(tokenAddress)}`)
    }

    //Set up things on L2 for this token

    L2ERC20 = await Factory__L2ERC20.deploy(
      (hre as any).deployConfig.L2StandardBridgeAddress,
      tokenAddress,
      //((hre as any).deployConfig.network === 'local' || token.symbol === 'TEST' ) ? L1ERC20.address : token.address,
      token.name,
      token.symbol,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2ERC20.deployTransaction.wait()
    
    const L2ERC20DeploymentSubmission: DeploymentSubmission = {
      ...L2ERC20,
      receipt: L2ERC20.receipt,
      address: L2ERC20.address,
      abi: L2ERC20.abi,
    };
    await hre.deployments.save(`TK_L2${token.symbol}`, L2ERC20DeploymentSubmission)
    console.log(`🌕 ${chalk.red(`L2 ${token.name} was deployed to`)} ${chalk.green(L2ERC20.address)}`)
  }
}

deployFn.tags = ['L1ERC20', 'test']

export default deployFn
