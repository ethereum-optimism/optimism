export class InvalidAddressError extends Error {
  constructor() {
    super(
      'An address was specified which is not a valid hex string of 20 bytes.'
    )
  }
}

export class InvalidBytesConsumedError extends Error {
  constructor() {
    super(
      "The specified bytes consumed does not match the opcode's actual consumed bytes."
    )
  }
}

export class UnsupportedOpcodeError extends Error {
  constructor() {
    super('Transpiler currently does not support the specified opcode.')
  }
}
