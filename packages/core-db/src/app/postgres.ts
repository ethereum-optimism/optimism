/* External Imports */
import {getLogger, logError} from '@eth-optimism/core-utils'

/* Internal Imports */
import {RDB, Row} from '../types/db'

const log = getLogger('postgres-db')
const Pool = require('pg-pool')

export class PostgresDB implements RDB {
  private readonly pool

  private constructor(
    host: string,
    port: number,
    user: string,
    password: string,
    database: string,
    poolSize: number = 10,
    ssl: boolean = false
  ) {
    this.pool = new Pool({
      database,
      user,
      password,
      host,
      port,
      ssl,
      max: poolSize,
      idleTimeoutMillis: 1000,
      connectionTimeoutMillis: 1000,
    })
  }

  public async select(query: string, client?: any): Promise<Row[]> {
    const c = client || await this.pool.connect()
    try {
      const res = await c.query(query)
      return res.rows
    } catch (e) {
      logError(log, `Error executing query ${query}!`, e)
      throw e
    } finally {
      if (!client) {
        c.release()
      }
    }
  }

  public async execute(query: string, client?: any): Promise<void> {
    const c = client || await this.pool.connect()
    try {
      // TODO: we can return IDs from here on inserts if we want this. Right now it doesn't matter.
      await c.query(query)
    } catch (e) {
      logError(log, `Error executing query ${query}!`, e)
      throw e
    } finally {
      if (!client) {
        c.release()
      }
    }
  }

  public async startTransaction(): Promise<any> {
    const client = await this.pool.connect()
    try {
      await client.query('BEGIN')
      return client
    } catch (e) {
      logError(log, `Error creating a transaction`, e)
      throw e
    } finally {
      client.release()
    }
  }

  public async commit(client: any): Promise<void> {
    try {
      await client.query('COMMIT')
    } catch (e) {
      logError(log, `Error committing transaction!`, e)
      throw e
    } finally {
      client.release()
    }
  }

  public async rollback(client: any): Promise<void> {
    try {
      await client.query('ROLLBACK')
    } catch (e) {
      logError(log, `Error rolling back transaction!`, e)
      throw e
    } finally {
      client.release()
    }
  }
}