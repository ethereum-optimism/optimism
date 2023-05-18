import { BigNumberish, ethers } from 'ethers'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

export type ModuleConfig = {
  authModuleDeploymentName: string
} & ContractModuleConfig

export interface ContractModuleConfig {
  ttl: BigNumberish
  amount: BigNumberish
  name: string
  enabled: boolean
}

export interface FaucetModuleConfigs {
  [name: string]: ModuleConfig
}

export enum Time {
  SECOND = 1,
  MINUTE = 60 * Time.SECOND,
  HOUR = 60 * Time.MINUTE,
  DAY = 24 * Time.HOUR,
  WEEK = 7 * Time.DAY,
}

export const getModuleConfigs = async (
  hre: HardhatRuntimeEnvironment
): Promise<Required<FaucetModuleConfigs>> => {
  let configs: FaucetModuleConfigs
  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    configs = require(`../../config/faucet/${hre.network.name}.ts`).default
  } catch (err) {
    throw new Error(
      `error while loading faucet module configs for network: ${hre.network.name}, ${err}`
    )
  }

  return configs
}

export const isSameConfig = (
  a: ContractModuleConfig,
  b: ContractModuleConfig
): boolean => {
  return (
    a.name === b.name &&
    ethers.BigNumber.from(a.amount).eq(b.amount) &&
    a.enabled === b.enabled &&
    ethers.BigNumber.from(a.ttl).eq(b.ttl)
  )
}
