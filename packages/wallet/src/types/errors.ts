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
