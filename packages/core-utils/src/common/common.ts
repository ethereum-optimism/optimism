export const assert = (condition: () => boolean, reason?: string) => {
  try {
    if (condition() === false) {
      throw new Error(`Assertion failed: ${reason}`)
    }
  } catch (err) {
    throw new Error(`Assertion failed: ${reason}\n${err}`)
  }
}
