/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory} from 'ethers'
import chalk from 'chalk';

import L1MessageJson from '../artifacts/contracts/test-helpers/Message/L1Message.sol/L1Message.json'
import L2MessageJson from '../artifacts-ovm/contracts/test-helpers/Message/L2Message.sol/L2Message.json'

let Factory__L1Message: ContractFactory
let Factory__L2Message: ContractFactory

let L1Message: Contract
let L2Message: Contract

const deployFn: DeployFunction = async (hre) => {

    Factory__L1Message = new ContractFactory(
      L1MessageJson.abi,
      L1MessageJson.bytecode,
      (hre as any).deployConfig.deployer_l1
    )

    Factory__L2Message = new ContractFactory(
      L2MessageJson.abi,
      L2MessageJson.bytecode,
      (hre as any).deployConfig.deployer_l2
    )

    const OVM_L1CrossDomainMessengerFastAddress = await (hre as any).deployConfig.addressManager.getAddress(
      'Proxy__OVM_L1CrossDomainMessengerFast'
    )

    L1Message = await Factory__L1Message.deploy(
      (hre as any).deployConfig.l1MessengerAddress,
      OVM_L1CrossDomainMessengerFastAddress
    )
    await L1Message.deployTransaction.wait()
    const L1MessageDeploymentSubmission: DeploymentSubmission = {
      ...L1Message,
      receipt: L1Message.receipt,
      address: L1Message.address,
      abi: L1MessageJson.abi,
    };
    await hre.deployments.save('L1Message', L1MessageDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L1 Message deployed to:')} ${chalk.green(L1Message.address)}`)

    L2Message = await Factory__L2Message.deploy(
      (hre as any).deployConfig.l2MessengerAddress,
      {gasLimit: 85450000}
    )
    await L2Message.deployTransaction.wait()
    const L2MessageDeploymentSubmission: DeploymentSubmission = {
      ...L2Message,
      receipt: L2Message.receipt,
      address: L2Message.address,
      abi: L2MessageJson.abi,
    };
    await hre.deployments.save('L2Message', L2MessageDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L2 Message deployed to:')} ${chalk.green(L2Message.address)}`)

    // Initialize L1 message
    const L1MessageTX = await L1Message.init(
      L2Message.address
    )
    await L1MessageTX.wait()
    console.log(`⭐️ ${chalk.blue('L1 Message initialized:')} ${chalk.green(L1MessageTX.hash)}`)

    // Initialize L2 message
    const L2MessageTX = await L2Message.init(
      L1Message.address,
      {gasLimit: 85450000}
    )
    await L2MessageTX.wait()
    console.log(`⭐️ ${chalk.blue('L2 Message initialized:')} ${chalk.green(L2MessageTX.hash)}`)

}

deployFn.tags = ['L1Message', 'L2Message', 'required']

export default deployFn
