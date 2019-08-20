export class CannotDecideError extends Error {
  constructor(message?: string) {
    super(message)
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export const handleCannotDecideError = (e): undefined => {
  if (!(e instanceof CannotDecideError)) {
    throw e
  }

  return undefined
}
