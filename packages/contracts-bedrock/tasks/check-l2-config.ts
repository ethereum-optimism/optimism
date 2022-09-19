import { task, types } from 'hardhat/config'
import { providers, Contract } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { predeploys } from '../src'

task('check-l2-config', 'Validate L2 config')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .setAction(async (args, hre) => {
    const l2Provider = new providers.JsonRpcProvider(args.l2ProviderUrl)

    const loadPredeploy = async (name: string): Promise<Contract> => {
      const artifact = await hre.artifacts.readArtifact(name)
      return new Contract(predeploys[name], artifact.abi, l2Provider)
    }

    const getContractAddress = async (name: string): Promise<string> => {
      const deployment = await hre.deployments.get(name)
      return deployment.address
    }

    // Verify that all predeploys have code.
    // TODO: Actually check that the predeploys have the expected code.
    for (const [name, address] of Object.entries(predeploys)) {
      const code = await l2Provider.getCode(address)
      if (code === '0x') {
        throw new Error(`Missing code for ${name}`)
      }
    }

    // Confirming that L2CrossDomainMessenger.otherMessenger() is set properly.
    const L2CrossDomainMessenger = await loadPredeploy('L2CrossDomainMessenger')
    const actualOtherMessenger = await getContractAddress(
      'L1CrossDomainMessengerProxy'
    )
    const expectedOtherMessenger = await L2CrossDomainMessenger.otherMessenger()
    if (expectedOtherMessenger !== actualOtherMessenger) {
      throw new Error(
        `L2CrossDomainMessenger otherMessenger not set correctly. Got ${actualOtherMessenger}, expected ${actualOtherMessenger}`
      )
    }

    // Confirming that L2StandardBridge.messenger() is set properly.
    const L2StandardBridge = await loadPredeploy('L2StandardBridge')
    const actualMessenger = await L2StandardBridge.messenger()
    const expectedMessenger = predeploys.L2CrossDomainMessenger
    if (expectedMessenger !== actualMessenger) {
      throw new Error(
        `L2StandardBridge messenger not set correctly. Got ${actualMessenger}, expected ${expectedMessenger}`
      )
    }

    // Confirming that L2StandardBridge.otherBridge() is set properly.
    const actualOtherBridge = await getContractAddress('L1StandardBridgeProxy')
    const expectedOtherBridge = await L2StandardBridge.otherBridge()
    if (expectedOtherBridge !== actualOtherBridge) {
      throw new Error(
        `L2StandardBridge otherBridge not set correctly. Got ${actualMessenger}, expected ${expectedOtherBridge}`
      )
    }

    // Confirming that OptimismMintableERC20Factory.bridge() is set properly.
    const OptimismMintableERC20Factory = await loadPredeploy(
      'OptimismMintableERC20Factory'
    )
    const actualBridge = await OptimismMintableERC20Factory.bridge()
    const expectedBridge = predeploys.L2StandardBridge
    if (expectedBridge !== actualBridge) {
      throw new Error(
        `OptimismMintableERC20Factory bridge not set correctly. Got ${actualBridge}, expected ${expectedBridge}`
      )
    }
  })
