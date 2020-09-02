export class InsufficientBalanceError extends Error {
  constructor() {
    super('Insufficient balance for transfer or swap!')
  }
}

export class NegativeAmountError extends Error {
  constructor() {
    super('Amounts transferred or swapped cannot be negative!')
  }
}

export class InvalidTransactionTypeError extends Error {
  constructor() {
    super('Invalid transaction type!')
  }
}

export class StateMachineCapacityError extends Error {
  constructor() {
    super('State machine is at capacity. No more addresses may be added!')
  }
}

export class InvalidTokenTypeError extends Error {
  constructor(type) {
    super(`Invalid token type received [${type}]. Must be 0 or 1.`)
  }
}

export class SignatureError extends Error {
  constructor() {
    super('Signature is invalid for the message in question.')
  }
}

export class UnexpectedBatchStatus extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export const isStateTransitionError = (error: Error): boolean => {
  return (
    error instanceof InsufficientBalanceError ||
    error instanceof NegativeAmountError ||
    error instanceof InvalidTransactionTypeError ||
    error instanceof StateMachineCapacityError ||
    error instanceof InvalidTokenTypeError ||
    error instanceof SignatureError
  )
}

export class NotSyncedError extends Error {
  constructor() {
    super(
      'The requested operation cannot be completed because this application is not synced.'
    )
  }
}

export class StateRootsMissingError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

/***************************
 * Batch Submission Errors *
 ***************************/

export class FutureRollupBatchNumberError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class FutureRollupBatchTimestampError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchBlockNumberTooOldError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchTimestampTooOldError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchSafetyQueueBlockNumberError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchSafetyQueueBlockTimestampError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchL1ToL2QueueBlockNumberError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchL1ToL2QueueBlockTimestampError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchOvmBlockNumberError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}

export class RollupBatchOvmTimestampError extends Error {
  constructor(msg: string) {
    super(msg)
  }
}
