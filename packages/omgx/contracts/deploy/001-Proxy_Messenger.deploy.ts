import { getContractFactory } from '@eth-optimism/contracts'
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory} from 'ethers'
import chalk from 'chalk'

require('dotenv').config()

import L1_MessengerJson from '../artifacts/contracts/OVM_L1CrossDomainMessengerFast.sol/OVM_L1CrossDomainMessengerFast.json'

let Factory__Proxy_L1_Messenger: ContractFactory

let Factory__L1_Messenger: ContractFactory

let Proxy_L1_Messenger: Contract

const deployFn: DeployFunction = async (hre) => {

    const addressManager = getContractFactory('Lib_AddressManager')
      .connect((hre as any).deployConfig.deployer_l1)
      .attach(process.env.ADDRESS_MANAGER_ADDRESS) as any

    Factory__L1_Messenger = new ContractFactory(
    L1_MessengerJson.abi,
    L1_MessengerJson.bytecode,
    (hre as any).deployConfig.deployer_l1
    )

    Factory__Proxy_L1_Messenger = getContractFactory(
        "Lib_ResolvedDelegateProxy",
        (hre as any).deployConfig.deployer_l1,
      )

    Proxy_L1_Messenger = await Factory__Proxy_L1_Messenger.deploy(
        addressManager.address,
        'OVM_L1CrossDomainMessengerFast',
    )

    await Proxy_L1_Messenger.deployTransaction.wait()

    const Proxy_L1_MessengerDeploymentSubmission: DeploymentSubmission = {
      ...Proxy_L1_Messenger,
      receipt: Proxy_L1_Messenger.receipt,
      address: Proxy_L1_Messenger.address,
      abi: Proxy_L1_Messenger.abi,
    };
    await hre.deployments.save('Proxy__OVM_L1CrossDomainMessengerFast', Proxy_L1_MessengerDeploymentSubmission)
    console.log(`🌕 ${chalk.red('Proxy__OVM_L1CrossDomainMessengerFast deployed to:')} ${chalk.green(Proxy_L1_Messenger.address)}`)

    const Proxy_L1_Messenger_Deployed = Factory__L1_Messenger.attach(Proxy_L1_Messenger.address)

    // initialize with address_manager
    const ProxyL1MessengerTX = await Proxy_L1_Messenger_Deployed.initialize(
      addressManager.address
    )
    console.log(`⭐️ ${chalk.blue('Proxy Fast L1 Messenger initialized:')} ${chalk.green(ProxyL1MessengerTX.hash)}`)

    const ProxyL1MessengerTXreg = await addressManager.setAddress(
        'Proxy__OVM_L1CrossDomainMessengerFast',
        Proxy_L1_Messenger.address
      )
      console.log(`⭐️ ${chalk.blue('Proxy Fast L1 Messager registered:')} ${chalk.green(ProxyL1MessengerTXreg.hash)}`)

}

deployFn.tags = ['Proxy_FastMessenger', 'required']

export default deployFn
