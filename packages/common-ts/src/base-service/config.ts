import { Validator } from './validators'

/**
 * Cleans a configuration object.
 *
 * @param config Configuration object to clean.
 * @param validators Mapping of config keys to validators.
 * @returns Cleaned configuration object.
 */
export const cleanConfig = (
  config: { [key: string]: any },
  validators: { [key: string]: Validator<any> }
) => {
  const cleaned = {}
  for (const [key, value] of Object.entries(config)) {
    const validator = validators[key] || validators[key.toLowerCase()]
    if (validator) {
      cleaned[key] = validator(value)
    } else {
      cleaned[key] = value
    }
  }
  return cleaned
}
