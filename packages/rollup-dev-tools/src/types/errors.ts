export class InvalidAddressError extends Error {
  constructor() {
    super(
      'An address was specified which is not a valid hex string of 20 bytes.'
    )
  }
}

export class OpcodeParseError extends Error {
  constructor() {
    super(
      'Attempted to parse an opcode representation, but was unable to match to the EVM spec.'
    )
  }
}

export class InvalidBytesConsumedError extends Error {
  constructor() {
    super(
      'Attempted to parse an opcode representation, but was unable to match to the EVM.'
    )
  }
}
