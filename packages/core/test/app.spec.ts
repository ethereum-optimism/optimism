import memdown from 'memdown'
import { PGCoreApp } from '../src/app'

const main = async () => {
  const pgCoreApp = new PGCoreApp({
    PLASMA_CHAIN_NAME: 'test',
    REGISTRY_ADDRESS: '0x0000000000000000000000000000000000000000',
    ETHEREUM_ENDPOINT: 'http://localhost:8545',
    BASE_DB_PATH: './testdb',
    DB_BACKEND: memdown,
  })
  await pgCoreApp.start()
}
main()
