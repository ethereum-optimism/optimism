export class MissingElementError extends Error {
  constructor(event: string) {
    super(`missing event: ${event}`)
  }
}
