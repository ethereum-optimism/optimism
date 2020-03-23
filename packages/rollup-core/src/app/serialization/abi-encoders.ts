/* External Imports */
import {add0x, getLogger} from '@eth-optimism/core-utils'

/* Internal Imports */
import {L2ToL1Message, RollupBlock, Transaction} from '../../types'
import {abi} from './common'

const log = getLogger('abiEncoders')

export const abiEncodeRollupBlock = (rollupBlock: RollupBlock): string => {
  // TODO: actually ABI encode blocks when they are solidified.
  return ''
}

export const abiEncodeTransaction = (transaction: Transaction): string => {
  // TODO: actually ABI encode transactions when they are solidified
  return ''
}


export const abiEncodeL2ToL1Message = (message: L2ToL1Message): string => {
  return abi.encode(['uint','address', 'bytes'], [
    add0x(message.nonce.toString('hex')),
    add0x(message.ovmSender),
    add0x(message.callData),
  ])
}