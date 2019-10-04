import { AddressBalance, StateChannelMessage } from './examples'
import { BigNumber, Message } from '../../types'
import { hexStrToBuf } from '../utils'

/**
 * Serializes the provided object to its canonical string representation.
 *
 * @param obj The object to serialize.
 * @returns The serialized object as a string.
 */
export const serializeObject = (obj: {}): string => {
  return JSON.stringify(obj)
}

/**
 * Deserializes the provided string into its object representation.
 * This assumes the string was serialized using the associated serializer.
 *
 * @param obj The string to deserialize.
 * @returns The deserialized object.
 */
export const deserializeObject = (obj: string): {} => {
  return JSON.parse(obj)
}

/**
 * Serializes the provided object to its canonical hex string representation.
 *
 * @param obj The object to serialize.
 * @returns The serialized object as a string.
 */
export const serializeObjectAsHexString = (obj: {}): string => {
  const stringified: string = JSON.stringify(obj)
  const asHexString = '0x' + Buffer.from(stringified, 'utf-8').toString('hex')
  return asHexString
}

/**
 * Deserializes the provided hex string into its object representation.
 * This assumes the string was serialized using the associated serializer.
 *
 * @param obj The string to deserialize.
 * @returns The deserialized object.
 */
export const deserializeObjectAsHexString = (obj: string): {} => {
  const asBuffer = hexStrToBuf(obj).slice(2)
  return JSON.parse(asBuffer.toString('utf-8'))
}

/**
 * Gets the canonical buffer representation of the provided object.
 *
 * @param obj The object
 * @returns The resulting Buffer
 */
export const objectToBuffer = (obj: {}): Buffer => {
  return Buffer.from(serializeObject(obj))
}

/**
 * Turns the provided StateChannelMessage into its canonical string representation.
 *
 * @param message The StateChannelMessage
 * @returns The resulting string
 */
export const stateChannelMessageToString = (
  message: StateChannelMessage
): string => {
  return serializeObject(message)
}

/**
 * Turns the provided Message into its canonical buffer representation.
 *
 * @param message The StateChannelMessage
 * @param messageSerializer: The serializer for turning the message's data object into a string
 * @returns The resulting Buffer
 */
export const messageToBuffer = (
  message: Message,
  messageSerializer: ({}) => string = (_) => '{}'
): Buffer => {
  return objectToBuffer({
    channelID: message.channelID.toString(),
    nonce: message.nonce,
    data: messageSerializer(message.data),
  })
}

/**
 * Turns the provided Message into its canonical string representation.
 *
 * @param message The StateChannelMessage
 * @param messageSerializer: The serializer for turning the message's data object into a string
 * @returns The resulting string
 */
export const messageToString = (
  message: Message,
  messageSerializer: ({}) => string = (_) => '{}'
): string => {
  return serializeObject({
    channelID: message.channelID.toString(),
    nonce: message.nonce,
    data: messageSerializer(message.data),
  })
}

/**
 * Deserializes the provided Buffer into the object it represents.
 *
 * @param buffer The buffer to be deserialized
 * @param messageDeserializer The deserializer for turning the buffer into the appropriate data object
 * @param functionParams The parameters (in addition to the string representation of the buffer) that the deserializer requires
 * @returns The resulting object
 */
export const deserializeBuffer = (
  buffer: Buffer,
  messageDeserializer: (string, any?) => any = (s) => JSON.parse(s),
  functionParams?: any
): any => {
  return messageDeserializer(buffer.toString(), functionParams)
}

/**
 * Deserializes the provided string into the object it represents.
 *
 * @param message The string to be deserialized
 * @param messageDeserializer The deserializer for turning the string into the appropriate data object
 * @param functionParams The parameters (in addition to the message string) that the deserializer requires
 * @returns The resulting object
 */
export const deserializeMessageString = (
  message: string,
  messageDeserializer: (string, any?) => any = (s) => JSON.parse(s),
  functionParams?: any
): any => {
  return messageDeserializer(message, functionParams)
}

/**
 * Deserializes the provided string into the Message it represents.
 *
 * @param message The string of the Message to be deserialized
 * @param dataDeserializer The deserializer for turning the data portion of the Message into the appropriate sub-message type
 * @returns The resulting Message
 */
export const deserializeMessage = (
  message: string,
  dataDeserializer: (string) => {} = (d) => d
): Message => {
  const parsedObject = deserializeObject(message)
  return {
    channelID: parsedObject['channelID'],
    nonce:
      'nonce' in parsedObject
        ? new BigNumber(parsedObject['nonce'])
        : undefined,
    data: dataDeserializer(parsedObject['data']),
  }
}

/**
 * Deserializes the provided string into a StateChannelMessage.
 *
 * @param message The string to convert into a StateChannelMessage.
 * @returns The resulting StateChannelMessage.
 */
export const stateChannelMessageDeserializer = (
  message: string
): StateChannelMessage => {
  const deserialized: {} = deserializeObject(message)
  const addressBalance: AddressBalance = {}
  Object.entries(deserialized['addressBalance']).forEach(
    ([address, balance]: [string, string]) => {
      addressBalance[address] = new BigNumber(balance)
    }
  )

  return {
    addressBalance,
  }
}
