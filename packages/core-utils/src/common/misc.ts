/**
 * Basic timeout-based async sleep function.
 *
 * @param ms Number of milliseconds to sleep.
 */
export const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve) => {
    setTimeout(() => {
      resolve(null)
    }, ms)
  })
}

/**
 * Returns a clone of the object.
 *
 * @param obj Object to clone.
 * @returns Clone of the object.
 */
export const clone = (obj: any): any => {
  if (typeof obj === 'undefined') {
    throw new Error(`Trying to clone undefined object`)
  }
  return { ...obj }
}

/**
 * Loads a variable from the environment and throws if the variable is not defined.
 *
 * @param name Name of the variable to load.
 * @returns Value of the variable as a string.
 */
export const reqenv = (name: string): string => {
  const value = process.env[name]
  if (value === undefined) {
    throw new Error(`missing env var ${name}`)
  }
  return value
}

/**
 * Loads a variable from the environment and returns a fallback if not found.
 *
 * @param name Name of the variable to load.
 * @param [fallback] Optional value to be returned as fallback.
 * @returns Value of the variable as a string, fallback or undefined.
 */
export const getenv = (name: string, fallback?: string): string | undefined => {
  return process.env[name] || fallback
}

/**
 * Returns true if the given string is a valid address.
 *
 * @param a First address to check.
 * @param b Second address to check.
 * @returns True if the given addresses match.
 */
export const compareAddrs = (a: string, b: string): boolean => {
  return a.toLowerCase() === b.toLowerCase()
}
