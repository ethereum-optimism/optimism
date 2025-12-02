//! This example decodes raw [Frame]s and reads them into a [Channel] and into a [SingleBatch].

use alloy_consensus::{SignableTransaction, TxEip1559, TxEnvelope};
use alloy_eips::eip2718::{Decodable2718, Encodable2718};
use alloy_primitives::{Address, BlockHash, Bytes, Signature, U256, hex};
use kona_genesis::RollupConfig;
use kona_protocol::{Batch, BlockInfo, Channel, Frame, SingleBatch, decompress_brotli};

fn main() {
    // Raw frame data taken from the `encode_channel` example.
    let first_frame = hex!(
        "60d54f49b71978b1b09288af847b11d200000000004d1b1301f82f0f6c3734f4821cd090ef3979d71a98e7e483b1dccdd525024c0ef16f425c7b4976a7acc0c94a0514b72c096d4dcc52f0b22dae193c70c86d0790a304a08152c8250031d091063ea000"
    );
    let second_frame = hex!(
        "60d54f49b71978b1b09288af847b11d2000100000046b00d00005082edde7ccf05bded2004462b5e80e1c42cd08e307f5baac723b22864cc6cd01ddde84efc7c018d7ada56c2fa8e3c5bedd494c3a7a884439d5771afcecaf196cb3801"
    );

    // Decode the raw frames.
    let decoded_first = Frame::decode(&first_frame).expect("decodes frame").1;
    let decoded_second = Frame::decode(&second_frame).expect("decodes frame").1;

    // Create a channel.
    let id = decoded_first.id;
    let open_block = BlockInfo::default();
    let mut channel = Channel::new(id, open_block);

    // Add the frames to the channel.
    let l1_inclusion_block = BlockInfo::default();
    channel.add_frame(decoded_first, l1_inclusion_block).expect("adds frame");
    channel.add_frame(decoded_second, l1_inclusion_block).expect("adds frame");

    // Get the frame data from the channel.
    let frame_data = channel.frame_data().expect("some frame data");
    println!("Frame data: {}", hex::encode(&frame_data));

    // Decompress the frame data with brotli.
    let config = RollupConfig::default();
    let max = config.max_rlp_bytes_per_channel(open_block.timestamp) as usize;
    let decompressed = decompress_brotli(&frame_data, max).expect("decompresses brotli");
    println!("Decompressed frame data: {}", hex::encode(&decompressed));

    // Decode the single batch from the decompressed data.
    let batch = Batch::decode(&mut decompressed.as_slice(), &config).expect("batch decodes");
    assert_eq!(
        batch,
        Batch::Single(SingleBatch {
            parent_hash: BlockHash::ZERO,
            epoch_num: 1,
            epoch_hash: BlockHash::ZERO,
            timestamp: 1,
            transactions: example_transactions(),
        })
    );

    println!("Successfully decoded frames into a Batch");
}

fn example_transactions() -> Vec<Bytes> {
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
