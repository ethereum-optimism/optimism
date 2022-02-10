// Use this file for simple types that aren't necessarily associated with a specific project or
// package. Often used for alias types like Address = string.

export interface Signature {
  r: string
  s: string
  v: number
}
export type Bytes32 = string
export type Uint16 = number
export type Uint8 = number
export type Uint24 = number
export type Address = string
