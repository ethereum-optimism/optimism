export class InvalidAddressError extends Error {
  constructor(msg?: string) {
    super(
      msg ||
        'An address was specified which is not a valid hex string of 20 bytes.'
    )
  }
}

export class InvalidBytesConsumedError extends Error {
  constructor(msg?: string) {
    super(
      msg ||
        "The specified bytes consumed does not match the opcode's actual consumed bytes."
    )
  }
}

export class UnsupportedOpcodeError extends Error {
  constructor(msg?: string) {
    super(msg || 'Transpiler currently does not support the specified opcode.')
  }
}

export class InvalidSubstitutionError extends Error {
  constructor(msg?: string) {
    super(
      msg ||
        'The configured replacements for the transpiler have resulted in invalid bytecode.'
    )
  }
}

export class InvalidInitcodeError extends Error {
  constructor(msg?: string) {
    super(msg || 'The specified initcode is invalid.')
  }
}

export class TranspilationErrors {
  public static readonly UNSUPPORTED_OPCODE: number = 0
  public static readonly OPCODE_NOT_WHITELISTED: number = 1
  public static readonly INVALID_BYTES_CONSUMED: number = 2
  public static readonly INVALID_SUBSTITUTION: number = 3
  public static readonly INVALID_INITCODE: number = 4
  public static readonly MISSING_CONSTANT_ERROR: number = 5
  public static readonly DETECTED_CONSTANT_OOB: number = 6
  public static readonly SUB_TRANSPILATION_ERROR: number = 7
  public static readonly MISSING_DEPLOYED_BYTECODE_ERROR: number = 8
  public static readonly CONSTRUCTOR_PARAMS_LOADER_DETECTION_FAILED: number = 9
}
