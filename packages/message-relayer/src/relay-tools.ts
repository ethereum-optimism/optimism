/* Imports: External */
import { Contract, ethers } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import { SentMessage } from './types'

/**
 * Parses a SentMessage event into a nice SentMessage struct.
 * @param event SentMessage event to parse.
 * @param l2BlockOffset The offset between the transaction index and the L2 block number.
 * @returns Parsed SentMessage event.
 */
export const parseSentMessageEvent = (
  event: ethers.Event,
  l2BlockOffset: number
): SentMessage => {
  // Event signature is `event SentMessage(bytes message)`
  // See here for reference:
  // https://github.com/ethereum-optimism/optimism/blob/2d352de8f257db2679b8ac6f953c146a8e99f05d/packages/contracts/contracts/optimistic-ethereum/iOVM/bridge/messaging/iAbs_BaseCrossDomainMessenger.sol#L14
  const message = event.args.message
  if (message === undefined) {
    throw new Error('event is not a SentMessage event')
  }

  // Message is an encoded call to `relayMessage`. We'll want to decode the components of the
  // message to simplify the process of inspecting the contents. We could theoretically leave this
  // decoding step out, but it's nice to have more information where possible.
  let decoded: {
    _target: string
    _sender: string
    _message: string
    _messageNonce: ethers.BigNumber
  }

  try {
    // Events are emitted on L2 here:
    // https://github.com/ethereum-optimism/optimism/blob/2d352de8f257db2679b8ac6f953c146a8e99f05d/packages/contracts/contracts/optimistic-ethereum/OVM/bridge/messaging/Abs_BaseCrossDomainMessenger.sol#L88
    // Encoding is performed here:
    // https://github.com/ethereum-optimism/optimism/blob/2d352de8f257db2679b8ac6f953c146a8e99f05d/packages/contracts/contracts/optimistic-ethereum/OVM/bridge/messaging/Abs_BaseCrossDomainMessenger.sol#L116-L122
    decoded = getContractInterface(
      'OVM_L2CrossDomainMessenger'
    ).decodeFunctionData('relayMessage', message) as any
  } catch (err) {
    throw new Error(
      `unable to parse SentMessage event from tx: ${event.transactionHash}`
    )
  }

  const checkNotUndefined = (property: string) => {
    if (decoded[property] === undefined) {
      throw new Error(
        `event is not a SentMessage event: ${property} is undefined`
      )
    }
  }

  // Make sure the component parts aren't undefined. In theory the above call to
  // `decodeFunctionData` shouldn't *allow* any of these fields to be undefined, but this is a
  // safety measure in case the function signature of `relayMessage` ever changes.
  checkNotUndefined('_target')
  checkNotUndefined('_sender')
  checkNotUndefined('_message')
  checkNotUndefined('_messageNonce')

  return {
    target: decoded._target,
    sender: decoded._sender,
    message: decoded._message,
    messageNonce: decoded._messageNonce.toNumber(),
    encodedMessage: message,
    encodedMessageHash: ethers.utils.keccak256(message),
    parentTransactionIndex: event.blockNumber - l2BlockOffset,
    parentTransactionHash: event.transactionHash,
  }
}

/**
 * Finds all messages sent through the L2CrossDomainMessenger between some start transaction height
 * (inclusive) and some end transaction height (exclusive).
 * @param ovmL2CrossDomainMessenger OVM_L2CrossDomainMessenger contract to query from.
 * @param l2BlockOffset The offset between the transaction index and the L2 block number.
 * @param startHeight Transaction height to start querying from (inclusive).
 * @param endHeight Transaction height to finish querying at (exclusive).
 * @returns All messages sent between the two transaction heights.
 */
export const getSentMessages = async (
  ovmL2CrossDomainMessenger: Contract,
  l2BlockOffset: number,
  startHeight: number,
  endHeight: number
): Promise<SentMessage[]> => {
  // Prevent some user errors. We could alternatively set endHeight = startHeight + 1 but I think
  // this is a little safer.
  if (endHeight <= startHeight) {
    throw new Error('end height must be greater than start height')
  }

  // Make sure the provided contract has the correct interface (mostly).
  if (ovmL2CrossDomainMessenger.filters.SentMessage === undefined) {
    throw new Error('SentMessage filter not found on provided contract')
  }

  // Find all SentMessage events between the two transaction heights.
  const events = await ovmL2CrossDomainMessenger.queryFilter(
    ovmL2CrossDomainMessenger.filters.SentMessage(),
    startHeight + l2BlockOffset,
    endHeight + l2BlockOffset - 1
  )

  // Parse each event into a SentMessage struct.
  const messages = events.map((event) => {
    return parseSentMessageEvent(event, l2BlockOffset)
  })

  // Make sure the messages are in ascending order.
  return messages.sort((a, b) => {
    return a.parentTransactionIndex - b.parentTransactionIndex
  })
}
