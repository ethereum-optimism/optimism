/* External Imports */
import { getContractDefinition } from '@eth-optimism/rollup-contracts'

export const ABI = {
  STATE_TRANSITIONER_ABI: getContractDefinition('StateTransitioner').abi,
  STATE_MANAGER_ABI: getContractDefinition('PartialStateManager').abi,
}
