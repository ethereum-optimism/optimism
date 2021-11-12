export class ErrMultipleSuccessfulRelays extends Error {
  constructor(hash: string) {
    super(`multiple successful relays for message with hash ${hash}`)
  }
}

export class ErrSentMessageMultipleTimes extends Error {
  constructor(hash: string) {
    super(`message with hash ${hash} sent multiple times`)
  }
}

export class ErrSentMessageNotFound extends Error {
  constructor(hash: string) {
    super(`message with hash ${hash} not found`)
  }
}

export class ErrTimeoutReached extends Error {
  constructor() {
    super(`timeout reached while polling`)
  }
}
