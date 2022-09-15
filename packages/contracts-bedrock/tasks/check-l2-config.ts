import { task, types } from 'hardhat/config'
import { providers, Contract } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { predeploys } from '../src'

const checkCode = async (provider: providers.JsonRpcProvider) => {
  for (const [name, address] of Object.entries(predeploys)) {
    const code = await provider.getCode(address)
    if (code === '0x') {
      throw new Error(`Missing code for ${name}`)
    }
  }
}

task('check-l2-config', 'Validate L2 config')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .setAction(async (args, hre) => {
    const { l2ProviderUrl } = args
    const l2Provider = new providers.JsonRpcProvider(l2ProviderUrl)

    await checkCode(l2Provider)

    const Artifact__L2CrossDomainMessenger = await hre.artifacts.readArtifact(
      'L2CrossDomainMessenger'
    )
    const Artifact__L2StandardBridge = await hre.artifacts.readArtifact(
      'L2StandardBridge'
    )
    const Artifact__OptimismMintableERC20Factory =
      await hre.artifacts.readArtifact('OptimismMintableERC20Factory')

    const L2CrossDomainMessenger = new Contract(
      predeploys.L2CrossDomainMessenger,
      Artifact__L2CrossDomainMessenger.abi,
      l2Provider
    )

    const Deployment__L1CrossDomainMessenger = await hre.deployments.get(
      'L1CrossDomainMessengerProxy'
    )
    const otherMessenger = await L2CrossDomainMessenger.otherMessenger()
    if (otherMessenger !== Deployment__L1CrossDomainMessenger.address) {
      throw new Error(
        `L2CrossDomainMessenger otherMessenger not set correctly. Got ${otherMessenger}, expected ${Deployment__L1CrossDomainMessenger.address}`
      )
    }

    const L2StandardBridge = new Contract(
      predeploys.L2StandardBridge,
      Artifact__L2StandardBridge.abi,
      l2Provider
    )

    const messenger = await L2StandardBridge.messenger()
    if (messenger !== predeploys.L2CrossDomainMessenger) {
      throw new Error(
        `L2StandardBridge messenger not set correctly. Got ${messenger}, expected ${predeploys.L2CrossDomainMessenger}`
      )
    }

    const Deployment__L1StandardBridge = await hre.deployments.get(
      'L1StandardBridgeProxy'
    )
    const otherBridge = await L2StandardBridge.otherBridge()
    if (otherBridge !== Deployment__L1StandardBridge.address) {
      throw new Error(
        `L2StandardBridge otherBridge not set correctly. Got ${otherBridge}, expected ${Deployment__L1StandardBridge.address}`
      )
    }

    const OptimismMintableERC20Factory = new Contract(
      predeploys.OptimismMintableERC20Factory,
      Artifact__OptimismMintableERC20Factory.abi,
      l2Provider
    )

    const bridge = await OptimismMintableERC20Factory.bridge()
    if (bridge !== predeploys.L2StandardBridge) {
      throw new Error(
        `OptimismMintableERC20Factory bridge not set correctly. Got ${bridge}, expected ${predeploys.L2StandardBridge}`
      )
    }
  })
