import { DBService } from '../../../src/services'
import { EphemDBProvider } from '../../../src/services/db/backends/ephem-db.provider'
import { config } from '../config.service'

config.set('DB_PROVIDER', EphemDBProvider)
export const dbservice = new DBService(config)
