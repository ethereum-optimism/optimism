import { DBManager } from '../../../interfaces'
import { BaseRunnable } from '../../common'
import { DefaultKeyManager } from '../../common/eth/key-manager'

export class KeyManagerHost extends BaseRunnable {
  private _keyManager: DefaultKeyManager

  constructor(private dbManager: DBManager) {
    super()
  }

  get keyManager(): DefaultKeyManager {
    return this._keyManager
  }

  public async onStart(): Promise<void> {
    const db = this.dbManager.create('keys')
    await db.open()
    this._keyManager = new DefaultKeyManager(db)
  }
}
