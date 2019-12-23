export interface TranspilationError {
  index: number
  error: number
  message: string
}

export interface TranspilationResultBase {
  succeeded: boolean
}

export interface ErroredTranspilation extends TranspilationResultBase {
  succeeded: false
  errors: TranspilationError[]
}

export interface SuccessfulTranspilation extends TranspilationResultBase {
  succeeded: true
  bytecode: Buffer
}

export type TranspilationResult = ErroredTranspilation | SuccessfulTranspilation
