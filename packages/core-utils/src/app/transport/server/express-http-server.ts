/* External Imports */
import bodyParser = require('body-parser')

/* Internal Imports */
import { HttpServer } from '../../../types'

/**
 * HTTP server that uses Express under the hood.
 */
export class ExpressHttpServer implements HttpServer {
  protected app
  private listening = false
  private server

  /**
   * Creates the server.
   * @param port Port to listen on.
   * @param hostname Hostname to listen on.
   */
  constructor(
    private port: number,
    private hostname: string,
    middleware?: Function[]
  ) {
    const express = require('express')
    this.app = express()
    // By default use bodyParser
    this.app.use(bodyParser.json({limit: '50mb'}))
    // Add any other middleware desired
    if (typeof middleware !== 'undefined') {
      for (const m of middleware) {
        this.app.use(m())
      }
    }
    this.initRoutes()
  }

  /**
   * Initializes any app routes.
   * App has no routes by default.
   */
  protected initRoutes(): void {
    return
  }

  /**
   * Starts the server.
   */
  public async listen(): Promise<void> {
    if (this.listening) {
      return
    }

    return new Promise<void>((resolve, reject) => {
      this.server = this.app.listen(this.port, this.hostname, () => {
        this.listening = true
        resolve()
      })
    })
  }

  /**
   * Stops the server.
   */
  public async close(): Promise<void> {
    if (!this.listening) {
      return
    }

    await this.server.close()
    this.listening = false
  }
}
