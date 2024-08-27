# PBS

Proposer builder seperation for Op Stack.

## How to run the devnet

### Running with default flashbots/op-geth builder image.

```shell
$ DEVNET_BUILDER=true make devnet-up
```

To additionally enable load testing through [tx-fuzz](https://github.com/MariusVanDerWijden/tx-fuzz), you can run the following command:

```shell
$ DEVNET_LOAD_TEST=true DEVNET_BUILDER=true make devnet-up
```

### Running with custom op-geth builder image.

You first need to build the op-geth docker image with builder API support.

```shell
$ git clone git@github.com:flashbots/op-geth.git
$ cd op-geth
$ docker build . -t <YOUR_OP_GETH_BUILDER_IMAGE>
```

Then you can run the devnet with the following command:

```shell
$ BUILDER_IMAGE=<YOUR_OP_GETH_BUILDER_IMAGE> DEVNET_BUILDER=true make devnet-up
```

## Configuration

These are the configuration options to enable PBS for the devnet.

### Sequencer

There are three flags that configure the sequencer to request payloads from the builder API endpoint:

| Flag                      | Description                                                         | Default Value |
|---------------------------|---------------------------------------------------------------------|---------------|
| `l2.builder.enabled`       | Enable the builder API request to get payloads built from the builder. | `false`       |
| `l2.builder.endpoint`      | The URL of the builder API endpoint.                                | `""`          |
| `l2.builder.timeout`       | The timeout for the builder API request.                            | `500ms`       |

### builder-op-node

The op-geth builder requires the op-node to publish the latest attributes as server-sent events in order to start building the payloads.

| Flag                        | Description                                                                     | Default Value |
|-----------------------------|---------------------------------------------------------------------------------|---------------|
| `sequencer.publish-attributes` | Set to true to enable the sequencer to publish attributes to the event stream. | `false`       |
| `eventstream.addr`          | The address of the eventstream server.                                           | `127.0.0.1`   |
| `eventstream.port`          | The port of the eventstream server.                                              | `9546`        |

### builder-op-geth

These are the builder flags to enable the builder service in op-geth:

| Flag                             | Description                                                                                  | Default Value |
|----------------------------------|----------------------------------------------------------------------------------------------|---------------|
| `builder`                        | Enable the builder service.                                                                  | `false`       |
| `builder.beacon_endpoints`       | The op-node address to get the payload attributes from. Should be set to `builder-op-node`.  | `""`          |
| `builder.block_retry_interval`   | The interval to retry building the payload.                                                  | `500ms`       |
| `builder.block_time`             | Block time of the network.                                                                   | `2s`          |
