import ganache from 'ganache-core';
import { BigNumber } from 'ethers/utils';
import { timeStamp } from 'console';
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

export class Ganache {
  private _options: GanacheServerOptions;
  private _server: GanacheServer;
  private _running: boolean;

  constructor(options: GanacheServerOptions = {}) {
    this._options = options;
    this._server = ganache.server(options);
  }

  public async start(): Promise<void> {
    this._options.port = this.port || await getPort();
    await this._server.listen(this._options.port);
    this._running = true;
  }

  public async stop(): Promise<void> {
    await this._server.close();
    this._running = false;
  }

  public get running(): boolean {
    return this._running;
  }

  public get port(): number | undefined {
    return this._options.port;
  }
}

