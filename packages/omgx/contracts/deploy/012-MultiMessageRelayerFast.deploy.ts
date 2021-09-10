import { getContractFactory } from '@eth-optimism/contracts'
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory} from 'ethers'
import chalk from 'chalk';

require('dotenv').config()

import L1_MultiMessageRelayerFastJson from '../artifacts/contracts/OVM_L1MultiMessageRelayerFast.sol/OVM_L1MultiMessageRelayerFast.json'

let Factory__L1_MultiMessageRelayerFast: ContractFactory

let L1_MultiMessageRelayerFast: Contract

const deployFn: DeployFunction = async (hre) => {

    const addressManager = getContractFactory('Lib_AddressManager')
      .connect((hre as any).deployConfig.deployer_l1)
      .attach(process.env.ADDRESS_MANAGER_ADDRESS) as any

    Factory__L1_MultiMessageRelayerFast = new ContractFactory(
      L1_MultiMessageRelayerFastJson.abi,
      L1_MultiMessageRelayerFastJson.bytecode,
      (hre as any).deployConfig.deployer_l1
    )

    L1_MultiMessageRelayerFast = await Factory__L1_MultiMessageRelayerFast.deploy(
      addressManager.address
    )

    await L1_MultiMessageRelayerFast.deployTransaction.wait()

    const L1_MultiMessageRelayerFastDeploymentSubmission: DeploymentSubmission = {
      ...L1_MultiMessageRelayerFast,
      receipt: L1_MultiMessageRelayerFast.receipt,
      address: L1_MultiMessageRelayerFast.address,
      abi: L1_MultiMessageRelayerFastJson.abi,
    };
    await hre.deployments.save('OVM_L1MultiMessageRelayerFast', L1_MultiMessageRelayerFastDeploymentSubmission)
    console.log(`🌕 ${chalk.red('OVM_L1MultiMessageRelayerFast deployed to:')} ${chalk.green(L1_MultiMessageRelayerFast.address)}`)

    //this will fail for non deployer account
    const L1MMRFastTXreg = await addressManager.setAddress(
      'OVM_L1MultiMessageRelayerFast',
      L1_MultiMessageRelayerFast.address
    )
    await L1MMRFastTXreg.wait()
    console.log(`⭐️ ${chalk.blue('OVM_L1MultiMessageRelayerFast registered:')} ${chalk.green(L1MMRFastTXreg.hash)}`)

    //register the fast batch message relayer too
    const FastBatchRelayerTXreg = await addressManager.setAddress(
      'OVM_L2BatchFastMessageRelayer',
      (hre as any).deployConfig.fastRelayerAddress
    )
    await FastBatchRelayerTXreg.wait()
    console.log(`⭐️ ${chalk.blue('OVM_L2BatchFastMessageRelayer Address registered:')} ${chalk.green(FastBatchRelayerTXreg.hash)}`)
}

deployFn.tags = ['MultiMessageRelayerFast', 'required']

export default deployFn
