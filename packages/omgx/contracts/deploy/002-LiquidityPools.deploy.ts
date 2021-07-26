/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory} from 'ethers'
import chalk from 'chalk';

import L1LiquidityPoolJson from '../artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'

let Factory__L1LiquidityPool: ContractFactory
let Factory__L2LiquidityPool: ContractFactory

let L1LiquidityPool: Contract
let L2LiquidityPool: Contract

const deployFn: DeployFunction = async (hre) => {

    Factory__L1LiquidityPool = new ContractFactory(
      L1LiquidityPoolJson.abi,
      L1LiquidityPoolJson.bytecode,
      (hre as any).deployConfig.deployer_l1
    )

    Factory__L2LiquidityPool = new ContractFactory(
      L2LiquidityPoolJson.abi,
      L2LiquidityPoolJson.bytecode,
      (hre as any).deployConfig.deployer_l2
    )
    // Deploy L2 liquidity pool
    console.log("Deploying...")
    L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
      (hre as any).deployConfig.l2MessengerAddress,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2LiquidityPool.deployTransaction.wait()
    const L2LiquidityPoolDeploymentSubmission: DeploymentSubmission = {
      ...L2LiquidityPool,
      receipt: L2LiquidityPool.receipt,
      address: L2LiquidityPool.address,
      abi: L1LiquidityPoolJson.abi,
    };
    await hre.deployments.save('L2LiquidityPool', L2LiquidityPoolDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L2LiquidityPool deployed to:')} ${chalk.green(L2LiquidityPool.address)}`)

    const OVM_L1CrossDomainMessengerFastAddress = await (hre as any).deployConfig.addressManager.getAddress(
      'Proxy__OVM_L1CrossDomainMessengerFast'
    )

    // Deploy L1 liquidity pool
    L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
      (hre as any).deployConfig.l1MessengerAddress,
      OVM_L1CrossDomainMessengerFastAddress
    )
    await L1LiquidityPool.deployTransaction.wait()
    const L1LiquidityPoolDeploymentSubmission: DeploymentSubmission = {
      ...L1LiquidityPool,
      receipt: L1LiquidityPool.receipt,
      address: L1LiquidityPool.address,
      abi: L2LiquidityPoolJson.abi,
    };
    await hre.deployments.save('L1LiquidityPool', L1LiquidityPoolDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L1LiquidityPool deployed to:')} ${chalk.green(L1LiquidityPool.address)}`)

    // Initialize L1 liquidity pool
    const L1LiquidityPoolTX = await L1LiquidityPool.init(
      /* userRewardFeeRate 3.5% */ 35,
      /* ownerRewardFeeRate 1.5% */ 15,
      L2LiquidityPool.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L1LiquidityPoolTX.wait()
    console.log(`⭐️ ${chalk.blue('L1 LP initialized:')} ${chalk.green(L1LiquidityPoolTX.hash)}`)

    // Initialize L2 liquidity pool
    const L2LiquidityPoolTX = await L2LiquidityPool.init(
      /* userRewardFeeRate 3.5% */ 35,
      /* ownerRewardFeeRate 1.5% */ 15,
      L1LiquidityPool.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2LiquidityPoolTX.wait()
    console.log(`⭐️ ${chalk.blue('L2 LP initialized:')} ${chalk.green(L2LiquidityPoolTX.hash)}`)

    let registerPoolETHTX = await L1LiquidityPool.registerPool(
      "0x0000000000000000000000000000000000000000",
      "0x4200000000000000000000000000000000000006",
    )
    await registerPoolETHTX.wait()

    registerPoolETHTX = await L2LiquidityPool.registerPool(
      "0x0000000000000000000000000000000000000000",
      "0x4200000000000000000000000000000000000006",
      {gasLimit: 800000, gasPrice: 0}
    )
    await registerPoolETHTX.wait()
    console.log(`L1 and L2 pools have registered ETH and oETH`)
}

deployFn.tags = ['L1LiquidityPool', 'L2LiquidityPool', 'required']

export default deployFn
