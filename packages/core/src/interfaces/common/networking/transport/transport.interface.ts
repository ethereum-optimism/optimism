/**
 * Client sends some data to a server over a transport layer.
 */
export interface Client<Request, Response> {
  /**
   * Sends a request over some transport layer.
   * @param data Data to send.
   * @returns the response of the request.
   */
  request(data: Request): Promise<Response>
}

/**
 * Server sends some data to a client over a transport layer.
 */
export interface Server<Response> {
  /**
   * Sends a response to a client.
   * @param data Data to send.
   */
  respond(data: Response): Promise<void>
}
