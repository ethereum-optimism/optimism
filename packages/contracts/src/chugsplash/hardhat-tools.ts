/* Imports: External */
import { HardhatRuntimeEnvironment } from 'hardhat/types'

/* Imports: Internal */
import { computeStorageSlots, getStorageLayout } from './storage'
import { ChugSplashConfig, parseChugSplashConfig } from './config'
import {
  ChugSplashAction,
  ChugSplashActionBundle,
  getChugSplashActionBundle,
} from './actions'

/**
 * Generates a ChugSplash action bundle from a config file.
 * @param hre Hardhat runtime environment, used to load artifacts + storage layouts.
 * @param config Config file to convert into a bundle.
 * @param env Environment variables to inject into the config file.
 * @returns Action bundle generated from the parsed config file.
 */
export const makeActionBundleFromConfig = async (
  hre: HardhatRuntimeEnvironment,
  config: ChugSplashConfig,
  env: any = {}
): Promise<ChugSplashActionBundle> => {
  const parsed = parseChugSplashConfig(config, env)

  const actions: ChugSplashAction[] = []
  for (const [contractName, contractConfig] of Object.entries(
    parsed.contracts
  )) {
    const artifact = hre.artifacts.readArtifactSync(contractConfig.source)
    const storageLayout = await getStorageLayout(hre, contractConfig.source)

    actions.push({
      target: contractConfig.address,
      code: artifact.deployedBytecode,
    })

    for (const slot of computeStorageSlots(
      storageLayout,
      contractConfig.variables
    )) {
      actions.push({
        target: contractConfig.address,
        key: slot.key,
        value: slot.val,
      })
    }
  }

  return getChugSplashActionBundle(actions)
}
