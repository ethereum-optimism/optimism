import { DeployConfigSpec } from '@eth-optimism/hardhat-deploy-config/dist/src/types'

/**
 * Defines the configuration for a deployment.
 */
export interface DeployConfig {
  /**
   * Dedicated Deterministic Deployer address (DDD).
   * When deploying authenticated deterministic smart contracts to the same address on various
   * chains, it's necessary to have a single root address that will initially own the contract and
   * later transfer ownership to the final contract owner. We call this address the DDD. We expect
   * the DDD to transfer ownership to the final contract owner very quickly after deployment.
   */
  ddd: string
}

/**
 * Specification for each of the configuration options.
 */
export const configSpec: DeployConfigSpec<DeployConfig> = {
  ddd: {
    type: 'address',
  },
}
