/**
 * Basic timeout-based async sleep function.
 * @param ms Number of milliseconds to sleep.
 */
export const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve, reject) => {
    setTimeout(() => {
      resolve(null)
    }, ms)
  })
}
