import { z } from 'zod'

/**
 * @internal
 * Default data type for attestations
 */
export const DEFAULT_DATA_TYPE = 'string' as const

/**
 * Zod validator for the DataType type
 * string | bytes | number | bool | address
 */
export const dataTypeOptionValidator = z
  .union([
    z.literal('string'),
    z.literal('bytes'),
    z.literal('number'),
    z.literal('bool'),
    z.literal('address'),
  ])
  .optional()
  .default('string').describe(`Zod validator for the DataType type
 string | bytes | number | bool | address`)

/**
 * Options for attestation data type
 */
export type DataTypeOption = z.infer<typeof dataTypeOptionValidator>
