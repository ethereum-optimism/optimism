# [server/server.go](./server.go)

Simple HTTP server package. It defines functions and types to handle JSON responses, response logging, and error recovery.

## Overview

- `RespondWithError` is a utility function to respond with an error message and HTTP status code. It uses the RespondWithJSON function to send the response.

- `RespondWithJSON` is a utility function that marshals the response payload into JSON and writes it to the HTTP response.

- `responseWriter` is a type that wraps an http.ResponseWriter to allow capturing the HTTP status code that was written. This is useful for logging.

- `wrapResponseWriter` is a simple function to wrap an http.ResponseWriter in a responseWriter.

- `LoggingMiddleware` is a middleware function for an HTTP server. It's used to wrap an HTTP handler to add logging for each request. It logs the HTTP method, path, status code, and how long the request took to process.

- The middleware also logs any panics that might occur during the request handling and recovers from them, to prevent the entire server from going down.

- The `metrics.RecordHTTPRequest` and `metrics.RecordHTTPResponse` functions are used to record metrics about the HTTP requests and responses for monitoring purposes.

- The logger parameter is an instance of a logger from the go-ethereum package, used to log various information.

