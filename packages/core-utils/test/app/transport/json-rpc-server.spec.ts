import '../../setup'

import { SimpleServer } from '../../../src/app/transport/server'
import { SimpleClient } from '../../../src/app/transport/client'

describe('Simple JSON RPC Server', () => {
  it('should send a request to a server and then respond', async () => {
    // Set up a server with a single method "greeter"
    const greeter = (name) => {
      return 'Hello ' + name
    }
    const server = new SimpleServer(
      {
        greeter,
      },
      'localhost',
      3000
    )
    await server.listen()

    // Set up a client which will call greeter with the name "World!"
    const client = new SimpleClient('http://127.0.0.1:3000')
    const res = await client.handle('greeter', 'World!')
    res.should.equal('Hello World!')

    // Close the server
    await server.close()
  })
})
