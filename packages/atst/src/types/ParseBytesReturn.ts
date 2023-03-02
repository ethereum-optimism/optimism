import { BigNumber } from 'ethers'
import { Address } from 'wagmi'

import { DataTypeOption } from './DataTypeOption'
import { WagmiBytes } from './WagmiBytes'

/**
 * @internal
 * Returns the correct typescript type of a DataOption
 */
export type ParseBytesReturn<T extends DataTypeOption> = T extends 'bytes'
  ? WagmiBytes
  : T extends 'number'
  ? BigNumber
  : T extends 'address'
  ? Address
  : T extends 'bool'
  ? boolean
  : T extends 'string'
  ? string
  : never
