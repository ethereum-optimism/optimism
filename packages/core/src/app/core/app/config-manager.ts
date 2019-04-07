import { stringify, jsonify } from '../../common'
import { ConfigManager } from '../../../interfaces'

/**
 * Simple config manager that stores configuration values
 * in memory as a jsonified object.
 */
export class SimpleConfigManager implements ConfigManager {
  private config: Record<string, any>

  constructor(config: Record<string, any> = {}) {
    this.config = { ...config }
  }

  /**
   * Queries a value from the config.
   * @param key Key to query.
   * @returns the config value.
   */
  public get(key: string): any {
    if (!(key in this.config)) {
      throw new Error('Key not found in configuration.')
    }

    const value = this.config[key]
    return jsonify(value)
  }

  /**
   * Sets a value in the config.
   * @param key Key to set.
   * @param value Value to set.
   */
  public put(key: string, value: any): void {
    const parsed = stringify(value)
    this.config[key] = parsed
  }
}
