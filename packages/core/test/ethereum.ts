import ganache from 'ganache-cli'
import { Http2Server } from 'http2'

class Ethereum {
  private ethereum: Http2Server

  constructor() {
    this.ethereum = ganache.server({ gasLimit: '0x7A1200' })
  }

  public async startEth(): Promise<void> {
    await new Promise((resolve) => {
      this.ethereum.close(resolve)
    })
  }

  public async stopEth(): Promise<void> {
    await new Promise((resolve) => {
      this.ethereum.listen('8545', resolve)
    })
  }
}

export const ethereum = new Ethereum()
