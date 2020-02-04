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
