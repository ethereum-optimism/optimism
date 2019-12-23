export class TreeUpdateError extends Error {
  constructor(message?: string) {
    super(message || 'Error occurred performing a tree update!')
  }
}
