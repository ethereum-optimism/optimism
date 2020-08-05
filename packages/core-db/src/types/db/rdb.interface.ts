export interface Row {
  [field: string]: any
}

/**
 * Base class for a relational database.
 */
export interface RDB {
  /**
   * Executes the provided query and returns its results.
   *
   * @param query The SQL query to execute.
   * @param client (optional) The client to use for the query
   * @returns The results of the query.
   */
  select(query: string, client?: any): Promise<Row[]>

  /**
   * Executes the provided query that expects no results
   *
   * @param query The SQL query to execute.
   * @param client (optional) The client to use for the query
   */
  execute(query: string, client?: any): Promise<void>

  /**
   * Starts a transaction.
   *
   * @returns The client to be used for all queries within the created transaction.
   */
  startTransaction(): Promise<any>

  /**
   * Commits a transaction that is open within the provided client.
   * As part of this call, the client will be closed.
   *
   * @param client The client.
   */
  commit(client: any): Promise<void>

  /**
   * Rolls back a transaction that is open within the provided client.
   * As part of this call, the client will be closed.
   *
   * @param client The client.
   */
  rollback(client: any): Promise<void>
}
