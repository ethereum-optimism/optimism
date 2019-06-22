export interface ServerBackend {
  app: any
  listen(): Promise<void>
}
