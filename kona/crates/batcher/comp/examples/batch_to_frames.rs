//! An example encoding and decoding a [SingleBatch].
//!
//! This example demonstrates EIP-2718 encoding a [SingleBatch]
//! through a [ChannelOut] and into individual [Frame]s.
//!
//! Notice, the raw batch is first _encoded_.
//! Once encoded, it is compressed into raw data that the channel is constructed with.
//!
//! The [ChannelOut] then outputs frames individually using the maximum frame size,
//! in this case hardcoded to 100, to construct the frames.
//!
//! Finally, once [Frame]s are built from the [ChannelOut], they are encoded and ready
//! to be batch-submitted to the data availability layer.

#[cfg(feature = "std")]
fn main() {
    use alloy_primitives::BlockHash;
    use kona_comp::{ChannelOut, CompressionAlgo, VariantCompressor};
    use kona_genesis::RollupConfig;
    use kona_protocol::{Batch, ChannelId, SingleBatch};

    // Use the example transaction
    let transactions = example_transactions();

    // Construct a basic `SingleBatch`
    let parent_hash = BlockHash::ZERO;
    let epoch_num = 1;
    let epoch_hash = BlockHash::ZERO;
    let timestamp = 1;
    let single_batch = SingleBatch { parent_hash, epoch_num, epoch_hash, timestamp, transactions };
    let batch = Batch::Single(single_batch);

    // Create a new channel.
    let id = ChannelId::default();
    let config = RollupConfig::default();
    let compressor: VariantCompressor = CompressionAlgo::Brotli10.into();
    let mut channel_out = ChannelOut::new(id, &config, compressor);

    // Add the compressed batch to the `ChannelOut`.
    channel_out.add_batch(batch).unwrap();

    // Output frames
    while channel_out.ready_bytes() > 0 {
        let frame = channel_out.output_frame(100).expect("outputs frame");
        println!("Frame: {}", alloy_primitives::hex::encode(frame.encode()));
        if channel_out.ready_bytes() <= 100 {
            channel_out.close();
        }
    }

    assert!(channel_out.closed);
    println!("Successfully encoded Batch to frames");
}

#[cfg(feature = "std")]
fn example_transactions() -> Vec<alloy_primitives::Bytes> {
    use alloy_consensus::{SignableTransaction, TxEip1559, TxEnvelope};
    use alloy_eips::eip2718::{Decodable2718, Encodable2718};
    use alloy_primitives::{Address, Signature, U256};

    let mut transactions = Vec::new();

    // First Transaction in the batch.
    let tx = TxEip1559 {
        chain_id: 10u64,
        nonce: 2,
        max_fee_per_gas: 3,
        max_priority_fee_per_gas: 4,
        gas_limit: 5,
        to: Address::left_padding_from(&[6]).into(),
        value: U256::from(7_u64),
        input: vec![8].into(),
        access_list: Default::default(),
    };
    let sig = Signature::test_signature();
    let tx_signed = tx.into_signed(sig);
    let envelope: TxEnvelope = tx_signed.into();
    let encoded = envelope.encoded_2718();
    transactions.push(encoded.clone().into());
    let mut slice = encoded.as_slice();
    let decoded = TxEnvelope::decode_2718(&mut slice).unwrap();
    assert!(matches!(decoded, TxEnvelope::Eip1559(_)));

    // Second transaction in the batch.
    let tx = TxEip1559 {
        chain_id: 10u64,
        nonce: 2,
        max_fee_per_gas: 3,
        max_priority_fee_per_gas: 4,
        gas_limit: 5,
        to: Address::left_padding_from(&[7]).into(),
        value: U256::from(7_u64),
        input: vec![8].into(),
        access_list: Default::default(),
    };
    let sig = Signature::test_signature();
    let tx_signed = tx.into_signed(sig);
    let envelope: TxEnvelope = tx_signed.into();
    let encoded = envelope.encoded_2718();
    transactions.push(encoded.clone().into());
    let mut slice = encoded.as_slice();
    let decoded = TxEnvelope::decode_2718(&mut slice).unwrap();
    assert!(matches!(decoded, TxEnvelope::Eip1559(_)));

    transactions
}

#[cfg(not(feature = "std"))]
fn main() {
    /* not implemented for no_std */
}
