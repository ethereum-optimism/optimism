/* External Imports */
import { BigNumber, getLogger } from '@eth-optimism/core-utils'

/* Internal imports */
import { L2ToL1Message, RollupBlock } from '../../types'
import { abi } from './common'

const log = getLogger('abiEncoders')

export const abiDecodeRollupBlock = (abiEncoded: string): RollupBlock => {
  // TODO: actually fill this out
  return {
    blockNumber: 1,
    stateRoot: '',
    transactions: [],
  }
}

export const abiDecodeL2ToL1Message = (abiEncoded: string): L2ToL1Message => {
  const [nonce, ovmSender, callData] = abi.decode(
    ['uint', 'address', 'bytes'],
    abiEncoded
  )

  return {
    nonce: new BigNumber(nonce),
    ovmSender,
    callData,
  }
}
