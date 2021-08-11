/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, ethers} from 'ethers'
import chalk from 'chalk';

import L1ProxyJson from '../artifacts/contracts/libraries/Lib_ResolvedDelegateProxy.sol/Lib_ResolvedDelegateProxy.json'
import L2ProxyJson from '../artifacts-ovm/contracts/libraries/Lib_ResolvedDelegateProxy.sol/Lib_ResolvedDelegateProxy.json'
import L1LiquidityPoolJson from '../artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'

let Factory__Proxy__L1LiquidityPool: ContractFactory
let Factory__Proxy__L2LiquidityPool: ContractFactory

let Proxy__L1LiquidityPool: Contract
let Proxy__L2LiquidityPool: Contract

const deployFn: DeployFunction = async (hre) => {

  Factory__Proxy__L1LiquidityPool = new ContractFactory(
    L1ProxyJson.abi,
    L1ProxyJson.bytecode,
    (hre as any).deployConfig.deployer_l1
  )

  Factory__Proxy__L2LiquidityPool = new ContractFactory(
    L2ProxyJson.abi,
    L2ProxyJson.bytecode,
    (hre as any).deployConfig.deployer_l2
  )

  // Deploy proxy contracts
  console.log(`💿 ${chalk.green("Deploying LP Proxy...")}`)

  const L1LiquidityPool = await (hre as any).deployments.get("L1LiquidityPool")
  const L2LiquidityPool = await (hre as any).deployments.get("L2LiquidityPool")
  const OVM_L1CrossDomainMessengerFastAddress = await (hre as any).deployConfig.addressManager.getAddress(
    'Proxy__OVM_L1CrossDomainMessengerFast'
  )

  Proxy__L1LiquidityPool = await Factory__Proxy__L1LiquidityPool.deploy(
    L1LiquidityPool.address
  )
  await Proxy__L1LiquidityPool.deployTransaction.wait()
  const Proxy__L1LiquidityPoolDeploymentSubmission: DeploymentSubmission = {
    ...Proxy__L1LiquidityPool,
    receipt: Proxy__L1LiquidityPool.receipt,
    address: Proxy__L1LiquidityPool.address,
    abi: Proxy__L1LiquidityPool.abi,
  };
  await hre.deployments.save('Proxy__L1LiquidityPool', Proxy__L1LiquidityPoolDeploymentSubmission)
  console.log(`🌕 ${chalk.red('Proxy__L1LiquidityPool deployed to:')} ${chalk.green(Proxy__L1LiquidityPool.address)}`)

  Proxy__L2LiquidityPool = await Factory__Proxy__L2LiquidityPool.deploy(
    L2LiquidityPool.address
  )
  await Proxy__L2LiquidityPool.deployTransaction.wait()
  const Proxy__L2LiquidityPoolDeploymentSubmission: DeploymentSubmission = {
    ...Proxy__L2LiquidityPool,
    receipt: Proxy__L2LiquidityPool.receipt,
    address: Proxy__L2LiquidityPool.address,
    abi: Proxy__L2LiquidityPool.abi,
  };
  await hre.deployments.save('Proxy__L2LiquidityPool', Proxy__L2LiquidityPoolDeploymentSubmission)
  console.log(`🌕 ${chalk.red('Proxy__L2LiquidityPool deployed to:')} ${chalk.green(Proxy__L2LiquidityPool.address)}`)

  Proxy__L1LiquidityPool = new ethers.Contract(
    Proxy__L1LiquidityPool.address,
    L1LiquidityPoolJson.abi,
    (hre as any).deployConfig.deployer_l1
  )

  const initL1LPTX = await Proxy__L1LiquidityPool.initialize(
    (hre as any).deployConfig.l1MessengerAddress,
    OVM_L1CrossDomainMessengerFastAddress,
    Proxy__L2LiquidityPool.address
  )
  await initL1LPTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L1LiquidityPool initialized:')} ${chalk.green(initL1LPTX.hash)}`)

  Proxy__L2LiquidityPool = new ethers.Contract(
    Proxy__L2LiquidityPool.address,
    L2LiquidityPoolJson.abi,
    (hre as any).deployConfig.deployer_l2
  )

  const initL2LPTX = await Proxy__L2LiquidityPool.initialize(
    (hre as any).deployConfig.l2MessengerAddress,
    Proxy__L1LiquidityPool.address,
    { gasLimit: 250000000}
  )
  await initL2LPTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L2LiquidityPool initialized:')} ${chalk.green(initL2LPTX.hash)}`)

  const registerL1LPETHTX = await Proxy__L1LiquidityPool.registerPool(
    "0x0000000000000000000000000000000000000000",
    "0x4200000000000000000000000000000000000006",
  )
  await registerL1LPETHTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L1LiquidityPool registered:')} ${chalk.green(registerL1LPETHTX.hash)}`)

  const registerL2LPETHTX = await Proxy__L2LiquidityPool.registerPool(
    "0x0000000000000000000000000000000000000000",
    "0x4200000000000000000000000000000000000006"//,
  )
  await registerL2LPETHTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L2LiquidityPool registered:')} ${chalk.green(registerL2LPETHTX.hash)}`)
}

deployFn.tags = ['Proxy__L1LiquidityPool', 'Proxy__L2LiquidityPool', 'required']

export default deployFn
