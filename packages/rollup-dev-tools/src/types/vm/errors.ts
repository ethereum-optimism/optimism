export class InvalidCALLStackError extends Error {
  constructor() {
    super('Stack before CALL is too small to correctly execute the CALL.')
  }
}
