export interface Row {
  data: any[]
  columns: { string: any }
}

export interface RDB {
  execute(query: string, withTransaction?: boolean): Promise<any>
  select(query: string, withNolock?: boolean): Promise<Row[]>

  begin(): Promise<void>
  commit(): Promise<void>
  rollback(): Promise<void>
}
