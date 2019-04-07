import { ConfigManager } from '../../../interfaces'
import { Process } from '../../common'
import { SimpleConfigManager } from './config-manager'

/**
 * Simple process wrapper that creates a new config manager instance.
 * Allows setting initial config via the constructor.
 */
export class SimpleConfigManagerProcess extends Process<ConfigManager> {
  /**
   * Creates the process.
   * @param [config] Optional initial config object.
   */
  constructor(private config?: Record<string, any>) {
    super()
  }

  /**
   * Creates the underlying config manager instance.
   */
  protected async onStart(): Promise<void> {
    this.subject = new SimpleConfigManager(this.config)
  }
}
