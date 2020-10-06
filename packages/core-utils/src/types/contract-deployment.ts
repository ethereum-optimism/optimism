import { Wallet } from 'ethers-v4'

/**
 * Deploys a contract and returns its deployed address.
 *
 * @param Wallet The Wallet to deploy from
 * @returns The deployed address as a hex string
 */
export type ContractDeploymentFunction = (w: Wallet) => Promise<string>
