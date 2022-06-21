export const getRangeEnd = (
  start: number,
  max: number,
  size: number
): number => {
  if (max < start) {
    throw new Error(`max must be >= start`)
  }

  return Math.min(start + size, max)
}

export const range = (start: number, length: number): Array<number> => {
  return [...Array(length).keys()].map((i) => {
    return i + start
  })
}
