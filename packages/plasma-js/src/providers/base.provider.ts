export interface BaseProvider {
  handle(method: string, params?: any[]): Promise<any>
}
