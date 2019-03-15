export type DBValue = string | object | number | boolean

export type DBResult = DBValue | DBValue[]

export interface DBObject {
  key: string
  value: DBValue
}

export interface DBOptions {
  [key: string]: any

  namespace: string
  id?: string
}

export interface BaseDBProvider {
  start(): Promise<void>
  get<T>(key: string, fallback?: T): Promise<T | DBResult>
  set(key: string, value: DBValue): Promise<void>
  delete(key: string): Promise<void>
  exists(key: string): Promise<boolean>
  findNextKey(key: string): Promise<string>
  bulkPut(objects: DBObject[]): Promise<void>
  push<T>(key: string, value: T): Promise<void>
}
