export class TransactionFormatError extends Error {
  constructor(message?: string) {
    super(message || 'The provided transaction was not formatted properly!')
  }
}

export class TransactionExecutionError extends Error {
  constructor(message?: string) {
    super(message || 'An unknown error occurred during transaction execution!')
  }
}

export class NotImplementedError extends Error {
  constructor(message?: string) {
    super(message || 'This feature is not implemented [yet]!')
  }
}

export class TransactionReceiptError extends Error {
  constructor(message?: string) {
    super(message || 'Error parsing transaction receipt!')
  }
}
