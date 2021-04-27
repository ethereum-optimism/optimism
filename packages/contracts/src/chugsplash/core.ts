/* External Imports */
import { Contract, ethers } from 'ethers'

/* Internal Imports */
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
  hash: string
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
    actions,
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
