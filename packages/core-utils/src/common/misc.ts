/**
 * Basic timeout-based async sleep function.
 *
 * @param ms Number of milliseconds to sleep.
 */
export const sleep = async (ms: number): Promise<void> => {
  return new Promise<void>((resolve, reject) => {
    setTimeout(() => {
      resolve(null)
    }, ms)
  })
}

// Returns a copy of an object
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

export const getenv = (name: string, fallback?: string): string | undefined => {
  return process.env[name] || fallback
}
