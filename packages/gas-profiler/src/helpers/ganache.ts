import ganache from 'ganache-core';
import { BigNumber } from 'ethers/utils';
const getPort = require('get-port');

interface GanacheServer {
  listen: (port: number) => Promise<void>,
  close: () => Promise<void>,
}

interface GanacheServerOptions {
  accounts?: Array<{
    secretKey: string,
    balance: string | BigNumber,
  }>,
  gasLimit?: number,
  port?: number,
}

/**
 * Helper class for managing ganache instances.
 */
export class Ganache {
  private _options: GanacheServerOptions;
  private _server: GanacheServer;
  private _running: boolean;

  constructor(options: GanacheServerOptions = {}) {
    this._options = options;
    this._server = ganache.server(options);
  }

  /**
   * Starts the ganache server.
   */
  public async start(): Promise<void> {
    this._options.port = this.port || await getPort();
    await this._server.listen(this._options.port);
    this._running = true;
  }

  /**
   * Stops the ganache server.
   */
  public async stop(): Promise<void> {
    await this._server.close();
    this._running = false;
  }

  /**
   * Server status indicator.
   * @returns `true` if the server is running, `false` otherwise.
   */
  public get running(): boolean {
    return this._running;
  }

  /**
   * Server port number.
   * @returns the current server port.
   */
  public get port(): number | undefined {
    return this._options.port;
  }
}

