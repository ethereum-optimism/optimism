import { Address } from '@wagmi/core'

import { DataTypeOption } from './DataTypeOption'

/**
 * The parameters for reading bulk attestations
 */
export interface AttestationReadParams {
  creator: Address
  about: Address
  key: string
  dataType?: DataTypeOption
  contractAddress?: Address
  allowFailure?: boolean
}
