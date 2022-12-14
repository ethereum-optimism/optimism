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

  /**
   * Name of the NFT in the Optimist contract.
   */
  optimistName: string

  /**
   * Symbol of the NFT in the Optimist contract.
   */
  optimistSymbol: string

  /**
   * Address of the priviledged attestor for the Optimist contract.
   */
  attestorAddress: string

  /**
   * Address of the owner of the proxies on L2. There will be a ProxyAdmin deployed as a predeploy
   * after bedrock, so the owner of proxies should be updated to that after the upgrade.
   * This currently is used as the owner of the nft related proxies.
   */
  l2ProxyOwnerAddress: string
}

/**
 * Specification for each of the configuration options.
 */
export const configSpec: DeployConfigSpec<DeployConfig> = {
  ddd: {
    type: 'address',
  },
  optimistName: {
    type: 'string',
  },
  optimistSymbol: {
    type: 'string',
  },
  attestorAddress: {
    type: 'address',
  },
  l2ProxyOwnerAddress: {
    type: 'address',
  },
}
