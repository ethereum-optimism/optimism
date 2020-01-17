import { hexStrToBuf, remove0x } from '../misc'

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
 * Gets the canonical buffer representation of the provided object.
 *
 * @param obj The object
 * @returns The resulting Buffer
 */
export const objectToBuffer = (obj: {}): Buffer => {
  return Buffer.from(serializeObject(obj))
}
