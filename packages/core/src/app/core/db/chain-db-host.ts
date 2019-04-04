import { MessageBus, DBManager, DB } from '../../../interfaces'
import { BaseRunnable } from '../../common'
import { DefaultMessageBus } from '../../common/app/message-bus'
import { DefaultDBManager } from '../../common/db/db-manager';
import { Service } from '@nestd/core';

@Service()
export class ChainDbHost extends BaseRunnable {
  private _db: DB

  constructor(private messageBus: DefaultMessageBus, private dbManager: DefaultDBManager) {
    super()
  }

  get db(): DB {
    return this._db
  }

  public async onStart(): Promise<void> {
    this.messageBus.on('contract:address', this.onAddressFound.bind(this))
  }

  private async onAddressFound(address: string): Promise<void> {
    this._db = this.dbManager.create(address)
    await this._db.open()
    this.messageBus.emit('chaindb:ready', address)
  }
}
