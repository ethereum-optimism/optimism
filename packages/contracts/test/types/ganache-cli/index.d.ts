declare module 'ganache-cli' {
  import { Http2Server } from 'http2'

  const ganache: Ganache
  export = ganache

  interface GanacheServerOptions {
    port: number | string
    gasLimit: string
  }

  interface Ganache {
    server(options: GanacheServerOptions): Http2Server
  }
}
