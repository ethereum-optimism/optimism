export class SlippageError extends Error {
  constructor() {
    super('Too much slippage in swap tx!')
  }
}

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

export class NotSyncedError extends Error {
  constructor() {
    super(
      'The requested operation cannot be completed because this application is not synced.'
    )
  }
}
