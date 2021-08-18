/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, ethers} from 'ethers'
import chalk from 'chalk';

import L1ProxyJson from '../artifacts/contracts/libraries/Lib_ResolvedDelegateProxy.sol/Lib_ResolvedDelegateProxy.json'
import L2ProxyJson from '../artifacts-ovm/contracts/libraries/Lib_ResolvedDelegateProxy.sol/Lib_ResolvedDelegateProxy.json'
import L1NFTBridgeJson from '../artifacts/contracts/bridges/OVM_L1NFTBridge.sol/OVM_L1NFTBridge.json'
import L2NFTBridgeJson from '../artifacts-ovm/contracts/bridges/OVM_L2NFTBridge.sol/OVM_L2NFTBridge.json'

let Factory__Proxy__L1NFTBridge: ContractFactory
let Factory__Proxy__L2NFTBridge: ContractFactory

let Proxy__L1NFTBridge: Contract
let Proxy__L2NFTBridge: Contract

const deployFn: DeployFunction = async (hre) => {

  Factory__Proxy__L1NFTBridge = new ContractFactory(
    L1ProxyJson.abi,
    L1ProxyJson.bytecode,
    (hre as any).deployConfig.deployer_l1
  )

  Factory__Proxy__L2NFTBridge = new ContractFactory(
    L2ProxyJson.abi,
    L2ProxyJson.bytecode,
    (hre as any).deployConfig.deployer_l2
  )

  // Deploy proxy contracts
  console.log(`💿 ${chalk.green("Deploying NFT Bridge Proxys...")}`)

  const L1NFTBridge = await (hre as any).deployments.get("L1NFTBridge")
  const L2NFTBridge = await (hre as any).deployments.get("L2NFTBridge")

  Proxy__L1NFTBridge = await Factory__Proxy__L1NFTBridge.deploy(
    L1NFTBridge.address
  )
  await Proxy__L1NFTBridge.deployTransaction.wait()
  const Proxy__L1NFTBridgeDeploymentSubmission: DeploymentSubmission = {
    ...Proxy__L1NFTBridge,
    receipt: Proxy__L1NFTBridge.receipt,
    address: Proxy__L1NFTBridge.address,
    abi: Proxy__L1NFTBridge.abi,
  };
  await hre.deployments.save('Proxy__L1NFTBridge', Proxy__L1NFTBridgeDeploymentSubmission)
  console.log(`🌕 ${chalk.red('Proxy__L1NFTBridge deployed to:')} ${chalk.green(Proxy__L1NFTBridge.address)}`)

  Proxy__L2NFTBridge = await Factory__Proxy__L2NFTBridge.deploy(
    L2NFTBridge.address
  )
  await Proxy__L2NFTBridge.deployTransaction.wait()
  const Proxy__L2NFTBridgeDeploymentSubmission: DeploymentSubmission = {
    ...Proxy__L2NFTBridge,
    receipt: Proxy__L2NFTBridge.receipt,
    address: Proxy__L2NFTBridge.address,
    abi: Proxy__L2NFTBridge.abi,
  };
  await hre.deployments.save('Proxy__L2NFTBridge', Proxy__L2NFTBridgeDeploymentSubmission)
  console.log(`🌕 ${chalk.red('Proxy__L2NFTBridge deployed to:')} ${chalk.green(Proxy__L2NFTBridge.address)}`)

  Proxy__L1NFTBridge = new ethers.Contract(
    Proxy__L1NFTBridge.address,
    L1NFTBridgeJson.abi,
    (hre as any).deployConfig.deployer_l1
  )

  const initL1NFTBridgeTX = await Proxy__L1NFTBridge.initialize(
    (hre as any).deployConfig.l1MessengerAddress,
    Proxy__L2NFTBridge.address
  )
  await initL1NFTBridgeTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L1NFTBridge initialized:')} ${chalk.green(initL1NFTBridgeTX.hash)}`)

  Proxy__L2NFTBridge = new ethers.Contract(
    Proxy__L2NFTBridge.address,
    L2NFTBridgeJson.abi,
    (hre as any).deployConfig.deployer_l2
  )

  const initL2NFTBridgeTX = await Proxy__L2NFTBridge.initialize(
    (hre as any).deployConfig.l2MessengerAddress,
    Proxy__L1NFTBridge.address,
    { gasLimit: 250000000}
  )
  await initL2NFTBridgeTX.wait()
  console.log(`⭐️ ${chalk.red('Proxy__L2NFTBridge initialized:')} ${chalk.green(initL2NFTBridgeTX.hash)}`)
}

deployFn.tags = ['Proxy__L1NFTBridge', 'Proxy__L2NFTBridge', 'required']

export default deployFn
