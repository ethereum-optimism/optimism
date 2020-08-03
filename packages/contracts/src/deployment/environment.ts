import { getLogger, logError } from '@eth-optimism/core-utils'

const log = getLogger('environment')

/**
 * Class to contain all environment variables referenced by the rollup full node
 * to consolidate access / updates and default values.
 */
export class Environment {
  public static getOrThrow<T>(
    fun: (defaultValue?: T) => T,
    defaultValue?: T,
    logValue: boolean = true
  ): T {
    const res = fun(defaultValue)
    if (res === undefined || (typeof res === 'number' && isNaN(res))) {
      throw Error(
        `Expected Environment variable not set. Error calling Environment.${fun.name}()`
      )
    }
    const lowerName: string = fun.name.toLowerCase()
    if (
      logValue &&
      lowerName.indexOf('password') < 0 &&
      lowerName.indexOf('private') < 0 &&
      lowerName.indexOf('mnemonic') < 0
    ) {
      log.info(`Environment: ${fun.name} = ${res}`)
    } else if (logValue) {
      log.info(
        `Environment: ${fun.name} is set (will not log value for security)`
      )
    }
    return res
  }

  // L1 Contract Params / Config
  public static l1ContractDeploymentPrivateKey(defaultValue?: string): string {
    return process.env.L1_CONTRACT_DEPLOYMENT_PRIVATE_KEY || defaultValue
  }
  public static l1ContractDeploymentMnemonic(defaultValue?: string): string {
    return process.env.L1_CONTRACT_DEPLOYMENT_MNEMONIC || defaultValue
  }

  // L1 Node Config -- Infura
  public static l1NodeInfuraNetwork(defaultValue?: string): string {
    return process.env.L1_NODE_INFURA_NETWORK || defaultValue
  }
  public static l1NodeInfuraProjectId(defaultValue?: string): string {
    return process.env.L1_NODE_INFURA_PROJECT_ID || defaultValue
  }
  // L1 Node Config -- URL
  public static l1NodeWeb3Url(defaultValue?: string): string {
    return process.env.L1_NODE_WEB3_URL || defaultValue
  }

  // Parameters / Config vars
  public static getL1ContractOwnerAddress(defaultValue?: string): string {
    return process.env.L1_CONTRACT_OWNER_ADDRESS || defaultValue
  }
  public static forceInclusionPeriodSeconds(defaultValue?: number): number {
    return process.env.FORCE_INCLUSION_PERIOD_SECONDS
      ? parseInt(process.env.FORCE_INCLUSION_PERIOD_SECONDS, 10)
      : defaultValue
  }
  public static sequencerAddress(defaultValue?: string): string {
    return process.env.L1_SEQUENCER_ADDRESS || defaultValue
  }
}
