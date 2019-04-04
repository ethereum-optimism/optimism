import Web3 from 'web3'
import ganache = require('ganache-cli')
import { Http2Server } from 'http2'

const port = '8545'
export const web3 = new Web3(`http://localhost:${port}`)

class Ethereum {
  private server: Http2Server
  private listening = false

  constructor() {
    this.server = ganache.server()
  }

  public async start(): Promise<void> {
    if (this.listening) {
      return
    }

    await new Promise((resolve) => {
      this.listening = true
      this.server.listen(port, resolve)
    })
  }

  public async stop(): Promise<void> {
    await new Promise((resolve) => {
      this.listening = false
      this.server.close(resolve)
    })
  }
}

export const ethereum = new Ethereum()
