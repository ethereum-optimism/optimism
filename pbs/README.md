# PBS

Proposer builder seperation for Op Stack.

## How to run the devnet

You first need to build the op-geth docker image with builder API support.

```shell
$ git clone git@github.com:flashbots/op-geth.git
$ cd op-geth
$ docker build . -t <YOUR_OP_GETH_BUILDER_IMAGE>
```

Then you can run the devnet with the following command:

```shell
$ DEVNET_BUILDER=true make devnet-up
```

To additionally enable load testing through [tx-fuzz](https://github.com/MariusVanDerWijden/tx-fuzz), you can run the following command:

```shell
$ DEVNET_LOAD_TEST=true DEVNET_BUILDER=true make devnet-up
```

If the BUILDER_OP_GETH_IMAGE is not set, the devnet will use the image from `flashbots/op-geth:latest`.

## Configuration

These are the configuration options to enable PBS for the devnet.

### Sequencer

There are three flags that congifure the sequencer to request payloads from builder API endpoint:

- `l2.builder.enabled`: Enable the builder API request to get payloads built from the builder.
- `l2.builder.endpoint`: The URL of the builder API endpoint.
- `l2.builder.timeout`: The timeout for the builder API request.

### builder-op-node

The op-geth builder requires the op-node to publish latest attributes as server sent events in order to start building the payloads.

- `sequencer.publish-attributes`: Set to true to enable the sequencer to publish attributes to the event stream.
- `eventstream.addr`: The address of the eventstream server.
- `eventstream.port`: The port of the eventstream server.
