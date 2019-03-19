/* External Imports */
import { Service, OnStart } from '@nestd/core'
import express = require('express')
import bodyParser = require('body-parser')
import cors = require('cors')
import { Express, Request, Response } from 'express'

/* Internal Imports */
import { ServerAlreadyStartedException } from '../exceptions'

@Service()
export class RpcServerService implements OnStart {
  private app: Express = express()
  private started: boolean = false

  public async onStart(): Promise<void> {
    if (this.started) {
      throw new ServerAlreadyStartedException()
    }

    this.app.use(bodyParser.json())
    this.app.use(cors())
    this.app.post('/api', this.handle.bind(this))
    this.app.listen(9821, '0.0.0.0')
  }

  private async handle(req: Request, res: Response): Promise<void> {
    // TODO: Use JsonRpcService and pipe to subdispatchers.
  }
}
