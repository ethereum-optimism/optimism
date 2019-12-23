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

export class TranspilationErrors {
  public static readonly UNSUPPORTED_OPCODE: number = 0
  public static readonly OPCODE_NOT_WHITELISTED: number = 1
  public static readonly INVALID_BYTES_CONSUMED: number = 2
}
