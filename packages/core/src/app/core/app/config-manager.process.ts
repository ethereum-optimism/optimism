import { ConfigManager } from '../../../interfaces'
import { Process } from '../../common'
import { DefaultConfigManager } from './config-manager'

export class DefaultConfigManagerProcess extends Process<ConfigManager> {
  constructor(private config: Record<string, any>) {
    super()
  }

  protected async onStart(): Promise<void> {
    this.subject = new DefaultConfigManager(this.config)
  }
}
