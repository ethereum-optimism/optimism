import { DataTypeOption } from '../types/DataTypeOption'

/**
 * @internal
 * Takes a datatype and returns the value casted to that type
 */
export const castAsDataType = (value: any, dataType: DataTypeOption) => {
  if (dataType === 'string') {
    return value
  } else if (dataType === 'number') {
    return Number(value)
  } else if (dataType === 'bool') {
    return Boolean(value)
  } else if (dataType === 'bytes') {
    return value
  } else if (dataType === 'address') {
    return value
  } else {
    throw new Error(`Unrecognized data type ${dataType satisfies never}`)
  }
}
