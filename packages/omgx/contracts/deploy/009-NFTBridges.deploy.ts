/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory} from 'ethers'
import chalk from 'chalk';

import L1NFTBridgeJson from '../artifacts/contracts/bridges/OVM_L1NFTBridge.sol/OVM_L1NFTBridge.json'
import L2NFTBridgeJson from '../artifacts-ovm/contracts/bridges/OVM_L2NFTBridge.sol/OVM_L2NFTBridge.json'

let Factory__L1NFTBridge: ContractFactory
let Factory__L2NFTBridge: ContractFactory

let L1NFTBridge: Contract
let L2NFTBridge: Contract

const deployFn: DeployFunction = async (hre) => {

    Factory__L1NFTBridge = new ContractFactory(
      L1NFTBridgeJson.abi,
      L1NFTBridgeJson.bytecode,
      (hre as any).deployConfig.deployer_l1
    )

    Factory__L2NFTBridge = new ContractFactory(
      L2NFTBridgeJson.abi,
      L2NFTBridgeJson.bytecode,
      (hre as any).deployConfig.deployer_l2
    )

    console.log("Deploying...")

    // Deploy L1 NFT Bridge
    L1NFTBridge = await Factory__L1NFTBridge.deploy()
    await L1NFTBridge.deployTransaction.wait()
    const L1NFTBridgeDeploymentSubmission: DeploymentSubmission = {
    ...L1NFTBridge,
    receipt: L1NFTBridge.receipt,
    address: L1NFTBridge.address,
    abi: L1NFTBridgeJson.abi,
    };
    await hre.deployments.save('L1NFTBridge', L1NFTBridgeDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L1NFTBridge deployed to:')} ${chalk.green(L1NFTBridge.address)}`)


    L2NFTBridge = await Factory__L2NFTBridge.deploy(
      {gasLimit: 250000000}
    )
    await L2NFTBridge.deployTransaction.wait()
    const L2NFTBridgeDeploymentSubmission: DeploymentSubmission = {
      ...L2NFTBridge,
      receipt: L2NFTBridge.receipt,
      address: L2NFTBridge.address,
      abi: L2NFTBridgeJson.abi,
    };
    await hre.deployments.save('L2NFTBridge', L2NFTBridgeDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L2NFTBridge deployed to:')} ${chalk.green(L2NFTBridge.address)}`)
}

deployFn.tags = ['L1NFTBridge', 'L2NFTBridge', 'required']

export default deployFn
