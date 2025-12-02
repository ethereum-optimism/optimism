/*
Package apis provides Go interfaces for most RPC / HTTP APIs used in the OP-Stack.

Every interface name ending with "Client" represents a client-binding:
this provides typing and methods exclusive to the client-side.
Client interfaces may use types not used for actual RPC transport.
E.g. a `uint64` argument instead of a `hexutil.Uint64`, or return an interface rather than a concrete type.
Client interfaces may also include additional methods, to enhance the calls to the server.
E.g. methods that adapt to call the right versioned server endpoint.

Every interface name ending with "Server" represents a server-binding:
this provides typing exclusive to the server-side.
E.g. a `hexutil.Uint64` argument instead of a `uint64`.
Servers should never provide methods not available in some form to the client.
These interfaces are used for type-checks of the server RPC backends.

Interface names not ending with "Client" or "Server" are generic, and can be used for both sides.
This is always preferred, to keep interfaces compatible, and perform type-checking across client/server.

Interfaces in this package are composed. When consuming interfaces as implementer,
always prefer to consume the smaller fitting interface, to increase compatibility.

Extension-interfaces, as popularized in the Golang FS design, may also be used on clients,
to expand scope only when necessary, and maintain simple defaults.
*/
package apis
