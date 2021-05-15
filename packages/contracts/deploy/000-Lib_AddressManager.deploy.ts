/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { registerAddress } from '../src/hardhat-deploy-ethers'
import { predeploys } from '../src/predeploys'

const deployFn: DeployFunction = async (hre) => {
  const { deploy } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()

  await deploy('Lib_AddressManager', {
    from: deployer,
    args: [],
    log: true,
  })

  await registerAddress({
    hre,
    name: 'OVM_L2CrossDomainMessenger',
    address: predeploys.OVM_L2CrossDomainMessenger,
  })

  await registerAddress({
    hre,
    name: 'OVM_Sequencer',
    address: (hre as any).deployConfig.ovmSequencerAddress,
  })

  await registerAddress({
    hre,
    name: 'OVM_Proposer',
    address: (hre as any).deployConfig.ovmProposerAddress,
  })

  await registerAddress({
    hre,
    name: 'OVM_L2BatchMessageRelayer',
    address: (hre as any).deployConfig.ovmRelayerAddress,
  })
}

deployFn.tags = ['Lib_AddressManager', 'required']

export default deployFn
