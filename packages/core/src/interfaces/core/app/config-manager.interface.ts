type ConfigKey = string | Buffer
type ConfigValue = string | Buffer

/**
 * Config handles storage of configuration values.
 */
export interface ConfigManager {
  /**
   * Gets a config value.
   * @param key to query.
   * @returns the value at that key.
   */
  get(key: ConfigKey): ConfigValue

  /**
   * Sets a config value.
   * @param key to set.
   * @param value to set the key to.
   */
  put(key: ConfigKey, value: ConfigValue): void
}
