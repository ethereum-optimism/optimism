# Batch Decoding Tool

The batch decoding tool is a utility to aid in debugging the batch submitter & the op-node
by looking at what batches were submitted on L1.

## Design Philosophy

The `batch_decoder` tool is designed to be simple & flexible. It offloads as much data analysis
as possible to other tools. It is built around manipulating JSON on disk. The first stage is to
fetch all transactions which are sent to a batch inbox address. Those transactions are decoded into
frames in that step & information about them is recorded. After transactions are fetched the frames
are re-assembled into channels in a second step that does not touch the network.


## Commands

### Fetch

`batch_decoder fetch` pulls all L1 transactions sent to the batch inbox address in a given L1 block
range and then stores them on disk to a specified path as JSON files where the name of the file is
the transaction hash.

### Reassemble

`batch_decoder reassemble` goes through all of the found frames in the cache & then turns them
into channels. It then stores the channels with metadata on disk where the file name is the Channel ID.
Each channel can contain multiple batches.

If the batch is span batch, `batch_decoder` derives span batch using `L2BlockTime`, `L2GenesisTime`, and `L2ChainID`.
These arguments can be provided to the binary using flags.

If the batch is a singular batch, `batch_decoder` does not derive and stores the batch as is.

### Force Close

`batch_decoder force-close` will create a transaction data that can be sent from the batcher address to
the batch inbox address which will force close the given channels. This will allow future channels to
be read without waiting for the channel timeout. It uses the results from `batch_decoder fetch` to
create the close transaction because the transaction it creates for a specific channel requires information
about if the channel has been closed or not. If it has been closed already but is missing specific frames
those frames need to be generated differently than simply closing the channel.


## JQ Cheat Sheet

`jq` is a really useful utility for manipulating JSON files.

```
# Pretty print a JSON file
jq . $JSON_FILE

# Print the number of valid & invalid transactions
jq .valid_data $TX_DIR/* | sort | uniq -c

# Select all transactions that have invalid data & then print the transaction hash
jq "select(.valid_data == false)|.tx.hash" $TX_DIR

# Select all channels that are not ready and then get the id and inclusion block & tx hash of the first frame.
jq "select(.is_ready == false)|[.id, .frames[0].inclusion_block, .frames[0].transaction_hash]"  $CHANNEL_DIR

# Show all of the frames in a channel without seeing the batches or frame data
jq 'del(.batches)|del(.frames[]|.frame.data)' $CHANNEL_FILE

# Show all batches (without timestamps) in a channel
jq '.batches|del(.[]|.Transactions)' $CHANNEL_FILE
```


## Roadmap

- Pull the batches out of channels & store that information inside the ChannelWithMetadata (CLI-3565)
	- Transaction Bytes used
	- Total uncompressed (different from tx bytes) + compressed bytes
- Invert ChannelWithMetadata so block numbers/hashes are mapped to channels they are submitted in (CLI-3560)
