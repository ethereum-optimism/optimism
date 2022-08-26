/* Imports: Internal */
import {DeployFunction} from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
    const {deploy} = hre.deployments
    const {deployer} = await hre.getNamedAccounts()
    const {deployConfig} = hre

    const portal = await hre.deployments.get('OptimismPortal')

    await deploy('L1CrossDomainMessenger', {
        from: deployer,
        args: [portal.address],
        log: true,
        waitConfirmations: deployConfig.deploymentWaitConfirmations,
    })

    const messenger = await hre.deployments.get('L1CrossDomainMessenger')

    const L1CrossDomainMessenger = await hre.ethers.getContractAt(
        'L1CrossDomainMessenger',
        messenger.address
    )

    const portalAddress = await L1CrossDomainMessenger.portal()
    if (portalAddress !== portal.address) {
        throw new Error('portal misconfigured')
    }
}

deployFn.tags = ['L1CrossDomainMessenger']

export default deployFn
