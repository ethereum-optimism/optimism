import hre from 'hardhat'
import { Contract, ethers } from 'ethers'
import * as dotenv from 'dotenv'

import { parseConfig } from './config'
import { computeStorageSlots, SolidityStorageLayout } from './storage'

enum ChugSplashActionType {
  SET_CODE,
  SET_STORAGE,
}

interface ChugSplashAction {
  type: ChugSplashActionType
  target: string
  data: string
}

export const getStorageLayout = async (
  hre: any, //HardhatRuntimeEnvironment,
  name: string
): Promise<SolidityStorageLayout> => {
  const { sourceName, contractName } = hre.artifacts.readArtifactSync(name)
  const buildInfo = await hre.artifacts.getBuildInfo(
    `${sourceName}:${contractName}`
  )
  const output = buildInfo.output.contracts[sourceName][contractName]

  if (!('storageLayout' in output)) {
    throw new Error(
      `Storage layout for ${name} not found. Did you forget to set the storage layout compiler option in your hardhat config? Read more: https://github.com/ethereum-optimism/smock#note-on-using-smoddit`
    )
  }

  return (output as any).storageLayout
}

export const getDeploymentBundle = async (
  hre: any, //HardhatRuntimeEnvironment,
  deploymentPath: string,
  deployerAddress: string
): Promise<{
  hash: string,
  actions: ChugSplashAction[]
}> => {
  const config = parseConfig(
    require(deploymentPath),
    deployerAddress,
    process.env
  )

  const actions: ChugSplashAction[] = []
  for (const [contractNickname, contractConfig] of Object.entries(
    config.contracts
  )) {
    const artifact = hre.artifacts.readArtifactSync(contractConfig.source)
    const storageLayout = await getStorageLayout(hre, contractConfig.source)

    // Push an action to deploy this contract.
    actions.push({
      type: ChugSplashActionType.SET_CODE,
      target: contractNickname,
      data: artifact.deployedBytecode,
    })

    // Push a `SET_STORAGE` action for each storage slot that we need to set.
    for (const slot of computeStorageSlots(
      storageLayout,
      contractConfig.variables
    )) {
      actions.push({
        type: ChugSplashActionType.SET_STORAGE,
        target: contractNickname,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [slot.key, slot.val]
        ),
      })
    }
  }

  return {
    hash: '0x' + 'FF'.repeat(32),
    actions
  }
}

export const createDeploymentManager = async (
  hre: any, //HardhatRuntimeEnvironment,
  owner: string
): Promise<Contract> => {
  const factory = await hre.ethers.getContractFactory('ChugSplashDeployer')
  const instance = await factory.deploy(owner)
  await instance.deployTransaction.wait()
  return instance
}

const main = async (hre: any) => {
  dotenv.config()

  const [owner] = await hre.ethers.getSigners()

  // 1. Create a ChugSplashDeployer
  const deployer = await createDeploymentManager(hre, await owner.getAddress())

  // 2. Generate the bundle of actions (SET_CODE or SET_STORAGE)
  const bundle = await getDeploymentBundle(
    hre,
    './deployments/old-deploy.json',
    deployer.address
  )

  // 3. Approve the bundle of actions.
  await deployer.approveTransactionBundle(
    bundle.hash,
    bundle.actions.length
  )

  // 4. Execute the bundle of actions.
  for (const action of bundle.actions) {
    console.log(`Executing chugsplash action`)
    console.log(`Target: ${action.target}`)
    console.log(`Type: ${action.type === 0 ? 'SET_CODE' : 'SET_STORAGE'}`)
    await deployer.executeAction(
      action.type,
      action.target,
      action.data,
      8_000_000 // TODO: how to handle gas?
    )
  }

  // 5. Verify the correctness of the deployment?
}

// misc improvements:
// want to minimize the need to perform unnecessary actions
// want to be able to perform multiple actions at the same time

main(hre)
