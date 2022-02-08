/**
 * TypeScript typings for bcoin's BCFG config parser (https://github.com/bcoin-org/bcfg)
 * This is NOT a complete set of typings, just what we use at Optimism at the moment.
 * We could consider expanding this into a full set of typings in the future.
 */
export interface Bcfg {
  /**
   * Loads configuration values from the environment. Must be called before environment variables
   * can be accessed with other methods like str(...) or uint(...).
   *
   * @param options Options to use when loading arguments.
   * @param options.env Boolean, whether or not to load from process.env.
   * @param options.argv Boolean, whether or not to load from process.argv.
   */
  load: (options: { env?: boolean; argv?: boolean }) => void

  /**
   * Returns the variable with the given name and casts it as a string. Queries from the
   * environment or from argv depending on which were loaded when load() was called.
   *
   * @param name Name of the variable to query.
   * @param defaultValue Optional default value if the variable does not exist.
   * @returns Variable cast to a string.
   */
  str: (name: string, defaultValue?: string) => string

  /**
   * Returns the variable with the given name and casts it as a uint. Will throw an error if the
   * variable cannot be cast into a uint. Queries from the environment or from argv depending on
   * which were loaded when load() was called.
   *
   * @param name Name of the variable to query.
   * @param defaultValue Optional default value if the variable does not exist.
   * @returns Variable cast to a uint.
   */
  uint: (name: string, defaultValue?: number) => number

  /**
   * Returns the variable with the given name and casts it as a bool. Will throw an error if the
   * variable cannot be cast into a bool. Queries from the environment or from argv depending on
   * which were loaded when load() was called.
   *
   * @param name Name of the variable to query.
   * @param defaultValue Optional default value if the variable does not exist.
   * @returns Variable cast to a bool.
   */
  bool: (name: string, defaultValue?: boolean) => boolean

  /**
   * Returns the variable with the given name and casts it as a ufloat. Will throw an error if the
   * variable cannot be cast into a ufloat. Queries from the environment or from argv depending on
   * which were loaded when load() was called.
   *
   * @param name Name of the variable to query.
   * @param defaultValue Optional default value if the variable does not exist.
   * @returns Variable cast to a ufloat.
   */
  ufloat: (name: string, defaultValue?: number) => number

  /**
   * Checks if the given variable exists.
   *
   * @param name Name of the variable to query.
   * @returns True if the variable exists, false otherwise.
   */
  has: (name: string) => boolean
}
