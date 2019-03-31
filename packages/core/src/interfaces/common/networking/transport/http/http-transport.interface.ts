import { Client, Server } from '../transport.interface'
import { HttpRequest, HttpResponse } from './http-message.interface'

export type HttpClient = Client<HttpRequest, HttpResponse>
export type HttpServer = Server<HttpResponse>
