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

export class UnsupportedFilterError extends Error {
  constructor(message?: string) {
    super(message || 'The provided filter is currently unsupported by the OVM')
  } 
}

export class RevertError extends Error {
  constructor(message?: string) {
    super(message || 'Revert: The provided transaction reverted.')
  }
}

export class RateLimitError extends Error {
  constructor(
    public readonly ipAddress: string,
    public readonly requestCount: number,
    public readonly limitPerPeriod: number,
    public readonly periodInMillis: number
  ) {
    super(
      `IP Address ${ipAddress} has made ${requestCount} requests in ${periodInMillis}ms, and only ${limitPerPeriod} are allowed in that timeframe.`
    )
  }
}

export class TransactionLimitError extends Error {
  constructor(
    public readonly address: string,
    public readonly transactionCount: number,
    public readonly limitPerPeriod: number,
    public readonly periodInMillis: number
  ) {
    super(
      `Address ${address} has attempted to send ${transactionCount} transactions in ${periodInMillis}ms, and only ${limitPerPeriod} are allowed in that timeframe.`
    )
  }
}

export class InvalidTransactionDesinationError extends Error {
  constructor(
    public readonly destinationAddress: string,
    public readonly validDestinationAddresses: string[]
  ) {
    super(
      `Invalid transaction destination ${destinationAddress}. The list of allowed addresses to send transactions to is ${JSON.stringify(
        validDestinationAddresses
      )}`
    )
  }
}
