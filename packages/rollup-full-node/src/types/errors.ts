export class TreeUpdateError extends Error {
  constructor(message?: string) {
    super(message || 'Error occurred performing a tree update!')
  }
}

export class UnsupportedMethodError extends Error {
  constructor(message?: string) {
    super(message || 'This method is not supported.')
  }
}

export class InvalidParametersError extends Error {
  constructor(message?: string) {
    super(
      message || 'The provided params are invalid for the call in question.'
    )
  }
}

export class RevertError extends Error {
  constructor(message?: string) {
    super(message || 'Revert: The provided transaction reverted.')
  }
}

export class RateLimitError extends Error {
  constructor(
    private readonly ipAddress: string,
    private readonly requestCount: number,
    private readonly limitPerPeriod: number,
    private readonly periodInMillis: number
  ) {
    super(
      `IP Address ${ipAddress} has made ${requestCount} requests in ${periodInMillis}ms, and only ${limitPerPeriod} are allowed in that timeframe.`
    )
  }
}

export class TransactionLimitError extends Error {
  constructor(
    private readonly address: string,
    private readonly transactionCount: number,
    private readonly limitPerPeriod: number,
    private readonly periodInMillis: number
  ) {
    super(
      `Address ${address} has attempted to send ${transactionCount} transactions in ${periodInMillis}ms, and only ${limitPerPeriod} are allowed in that timeframe.`
    )
  }
}

export class InvalidTransactionDesinationError extends Error {
  constructor(
    private readonly destinationAddress: string,
    private readonly validDestinationAddresses: string[]
  ) {
    super(
      `Invalid transaction destination ${destinationAddress}. The list of allowed addresses to send transactions to is ${JSON.stringify(
        validDestinationAddresses
      )}`
    )
  }
}
