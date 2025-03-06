//! Optimism-specific implementation and utilities for the executor

use crate::{error::L1BlockInfoError, OpBlockExecutionError};
use alloy_consensus::Transaction;
use alloy_primitives::{address, b256, hex, Address, Bytes, B256, U256};
use op_revm::{L1BlockInfo, OpSpecId};
use reth_execution_errors::BlockExecutionError;
use reth_optimism_forks::OpHardforks;
use reth_primitives_traits::BlockBody;
use revm::{primitives::HashMap, state::Bytecode, DatabaseCommit};
use tracing::trace;

/// The address of the create2 deployer
const CREATE_2_DEPLOYER_ADDR: Address = address!("0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2");

/// The codehash of the create2 deployer contract.
const CREATE_2_DEPLOYER_CODEHASH: B256 =
    b256!("0xb0550b5b431e30d38000efb7107aaa0ade03d48a7198a140edda9d27134468b2");

/// The raw bytecode of the create2 deployer contract.
const CREATE_2_DEPLOYER_BYTECODE: [u8; 1584] = hex!("6080604052600436106100435760003560e01c8063076c37b21461004f578063481286e61461007157806356299481146100ba57806366cfa057146100da57600080fd5b3661004a57005b600080fd5b34801561005b57600080fd5b5061006f61006a366004610327565b6100fa565b005b34801561007d57600080fd5b5061009161008c366004610327565b61014a565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200160405180910390f35b3480156100c657600080fd5b506100916100d5366004610349565b61015d565b3480156100e657600080fd5b5061006f6100f53660046103ca565b610172565b61014582826040518060200161010f9061031a565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe082820381018352601f90910116604052610183565b505050565b600061015683836102e7565b9392505050565b600061016a8484846102f0565b949350505050565b61017d838383610183565b50505050565b6000834710156101f4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f437265617465323a20696e73756666696369656e742062616c616e636500000060448201526064015b60405180910390fd5b815160000361025f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f437265617465323a2062797465636f6465206c656e677468206973207a65726f60448201526064016101eb565b8282516020840186f5905073ffffffffffffffffffffffffffffffffffffffff8116610156576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f437265617465323a204661696c6564206f6e206465706c6f790000000000000060448201526064016101eb565b60006101568383305b6000604051836040820152846020820152828152600b8101905060ff815360559020949350505050565b61014e806104ad83390190565b6000806040838503121561033a57600080fd5b50508035926020909101359150565b60008060006060848603121561035e57600080fd5b8335925060208401359150604084013573ffffffffffffffffffffffffffffffffffffffff8116811461039057600080fd5b809150509250925092565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000806000606084860312156103df57600080fd5b8335925060208401359150604084013567ffffffffffffffff8082111561040557600080fd5b818601915086601f83011261041957600080fd5b81358181111561042b5761042b61039b565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f011681019083821181831017156104715761047161039b565b8160405282815289602084870101111561048a57600080fd5b826020860160208301376000602084830101528095505050505050925092509256fe608060405234801561001057600080fd5b5061012e806100206000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063249cb3fa14602d575b600080fd5b603c603836600460b1565b604e565b60405190815260200160405180910390f35b60008281526020818152604080832073ffffffffffffffffffffffffffffffffffffffff8516845290915281205460ff16608857600060aa565b7fa2ef4600d742022d532d4747cb3547474667d6f13804902513b2ec01c848f4b45b9392505050565b6000806040838503121560c357600080fd5b82359150602083013573ffffffffffffffffffffffffffffffffffffffff8116811460ed57600080fd5b80915050925092905056fea26469706673582212205ffd4e6cede7d06a5daf93d48d0541fc68189eeb16608c1999a82063b666eb1164736f6c63430008130033a2646970667358221220fdc4a0fe96e3b21c108ca155438d37c9143fb01278a3c1d274948bad89c564ba64736f6c63430008130033");

/// The function selector of the "setL1BlockValuesEcotone" function in the `L1Block` contract.
const L1_BLOCK_ECOTONE_SELECTOR: [u8; 4] = hex!("440a5e20");

/// The function selector of the "setL1BlockValuesIsthmus" function in the `L1Block` contract.
const L1_BLOCK_ISTHMUS_SELECTOR: [u8; 4] = hex!("098999be");

/// Extracts the [`L1BlockInfo`] from the L2 block. The L1 info transaction is always the first
/// transaction in the L2 block.
///
/// Returns an error if the L1 info transaction is not found, if the block is empty.
pub fn extract_l1_info<B: BlockBody>(body: &B) -> Result<L1BlockInfo, OpBlockExecutionError> {
    let l1_info_tx = body
        .transactions()
        .first()
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::MissingTransaction))?;
    extract_l1_info_from_tx(l1_info_tx)
}

/// Extracts the [`L1BlockInfo`] from the L1 info transaction (first transaction) in the L2
/// block.
///
/// Returns an error if the calldata is shorter than 4 bytes.
pub fn extract_l1_info_from_tx<T: Transaction>(
    tx: &T,
) -> Result<L1BlockInfo, OpBlockExecutionError> {
    let l1_info_tx_data = tx.input();
    if l1_info_tx_data.len() < 4 {
        return Err(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::InvalidCalldata));
    }

    parse_l1_info(l1_info_tx_data)
}

/// Parses the input of the first transaction in the L2 block, into [`L1BlockInfo`].
///
/// Returns an error if data is incorrect length.
///
/// Caution this expects that the input is the calldata of the [`L1BlockInfo`] transaction (first
/// transaction) in the L2 block.
///
/// # Panics
/// If the input is shorter than 4 bytes.
pub fn parse_l1_info(input: &[u8]) -> Result<L1BlockInfo, OpBlockExecutionError> {
    // If the first 4 bytes of the calldata are the L1BlockInfoEcotone selector, then we parse the
    // calldata as an Ecotone hardfork L1BlockInfo transaction. Otherwise, we parse it as a
    // Bedrock hardfork L1BlockInfo transaction.
    if input[0..4] == L1_BLOCK_ISTHMUS_SELECTOR {
        parse_l1_info_tx_isthmus(input[4..].as_ref())
    } else if input[0..4] == L1_BLOCK_ECOTONE_SELECTOR {
        parse_l1_info_tx_ecotone(input[4..].as_ref())
    } else {
        parse_l1_info_tx_bedrock(input[4..].as_ref())
    }
}

/// Parses the calldata of the [`L1BlockInfo`] transaction pre-Ecotone hardfork.
pub fn parse_l1_info_tx_bedrock(data: &[u8]) -> Result<L1BlockInfo, OpBlockExecutionError> {
    // The setL1BlockValues tx calldata must be exactly 260 bytes long, considering that
    // we already removed the first 4 bytes (the function selector). Detailed breakdown:
    //   32 bytes for the block number
    // + 32 bytes for the block timestamp
    // + 32 bytes for the base fee
    // + 32 bytes for the block hash
    // + 32 bytes for the block sequence number
    // + 32 bytes for the batcher hash
    // + 32 bytes for the fee overhead
    // + 32 bytes for the fee scalar
    if data.len() != 256 {
        return Err(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::UnexpectedCalldataLength));
    }

    let l1_base_fee = U256::try_from_be_slice(&data[64..96])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BaseFeeConversion))?;
    let l1_fee_overhead = U256::try_from_be_slice(&data[192..224])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::FeeOverheadConversion))?;
    let l1_fee_scalar = U256::try_from_be_slice(&data[224..256])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::FeeScalarConversion))?;

    let mut l1block = L1BlockInfo::default();
    l1block.l1_base_fee = l1_base_fee;
    l1block.l1_fee_overhead = Some(l1_fee_overhead);
    l1block.l1_base_fee_scalar = l1_fee_scalar;

    Ok(l1block)
}

/// Updates the L1 block values for an Ecotone upgraded chain.
/// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
/// Params are expected to be in the following order:
///   1. _baseFeeScalar      L1 base fee scalar
///   2. _blobBaseFeeScalar  L1 blob base fee scalar
///   3. _sequenceNumber     Number of L2 blocks since epoch start.
///   4. _timestamp          L1 timestamp.
///   5. _number             L1 blocknumber.
///   6. _basefee            L1 base fee.
///   7. _blobBaseFee        L1 blob base fee.
///   8. _hash               L1 blockhash.
///   9. _batcherHash        Versioned hash to authenticate batcher by.
///
/// <https://github.com/ethereum-optimism/optimism/blob/957e13dd504fb336a4be40fb5dd0d8ba0276be34/packages/contracts-bedrock/src/L2/L1Block.sol#L136>
pub fn parse_l1_info_tx_ecotone(data: &[u8]) -> Result<L1BlockInfo, OpBlockExecutionError> {
    if data.len() != 160 {
        return Err(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::UnexpectedCalldataLength));
    }

    // https://github.com/ethereum-optimism/op-geth/blob/60038121c7571a59875ff9ed7679c48c9f73405d/core/types/rollup_cost.go#L317-L328
    //
    // data layout assumed for Ecotone:
    // offset type varname
    // 0     <selector>
    // 4     uint32 _basefeeScalar (start offset in this scope)
    // 8     uint32 _blobBaseFeeScalar
    // 12    uint64 _sequenceNumber,
    // 20    uint64 _timestamp,
    // 28    uint64 _l1BlockNumber
    // 36    uint256 _basefee,
    // 68    uint256 _blobBaseFee,
    // 100   bytes32 _hash,
    // 132   bytes32 _batcherHash,

    let l1_base_fee_scalar = U256::try_from_be_slice(&data[..4])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BaseFeeScalarConversion))?;
    let l1_blob_base_fee_scalar = U256::try_from_be_slice(&data[4..8]).ok_or({
        OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BlobBaseFeeScalarConversion)
    })?;
    let l1_base_fee = U256::try_from_be_slice(&data[32..64])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BaseFeeConversion))?;
    let l1_blob_base_fee = U256::try_from_be_slice(&data[64..96])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BlobBaseFeeConversion))?;

    let mut l1block = L1BlockInfo::default();
    l1block.l1_base_fee = l1_base_fee;
    l1block.l1_base_fee_scalar = l1_base_fee_scalar;
    l1block.l1_blob_base_fee = Some(l1_blob_base_fee);
    l1block.l1_blob_base_fee_scalar = Some(l1_blob_base_fee_scalar);

    Ok(l1block)
}

/// Updates the L1 block values for an Isthmus upgraded chain.
/// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
/// Params are expected to be in the following order:
///   1. _baseFeeScalar       L1 base fee scalar
///   2. _blobBaseFeeScalar   L1 blob base fee scalar
///   3. _sequenceNumber      Number of L2 blocks since epoch start.
///   4. _timestamp           L1 timestamp.
///   5. _number              L1 blocknumber.
///   6. _basefee             L1 base fee.
///   7. _blobBaseFee         L1 blob base fee.
///   8. _hash                L1 blockhash.
///   9. _batcherHash         Versioned hash to authenticate batcher by.
///  10. _operatorFeeScalar   Operator fee scalar
///  11. _operatorFeeConstant Operator fee constant
pub fn parse_l1_info_tx_isthmus(data: &[u8]) -> Result<L1BlockInfo, OpBlockExecutionError> {
    if data.len() != 172 {
        return Err(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::UnexpectedCalldataLength));
    }

    // https://github.com/ethereum-optimism/op-geth/blob/60038121c7571a59875ff9ed7679c48c9f73405d/core/types/rollup_cost.go#L317-L328
    //
    // data layout assumed for Ecotone:
    // offset type varname
    // 0     <selector>
    // 4     uint32 _basefeeScalar (start offset in this scope)
    // 8     uint32 _blobBaseFeeScalar
    // 12    uint64 _sequenceNumber,
    // 20    uint64 _timestamp,
    // 28    uint64 _l1BlockNumber
    // 36    uint256 _basefee,
    // 68    uint256 _blobBaseFee,
    // 100   bytes32 _hash,
    // 132   bytes32 _batcherHash,
    // 164   uint32 _operatorFeeScalar
    // 168   uint64 _operatorFeeConstant

    let l1_base_fee_scalar = U256::try_from_be_slice(&data[..4])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BaseFeeScalarConversion))?;
    let l1_blob_base_fee_scalar = U256::try_from_be_slice(&data[4..8]).ok_or({
        OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BlobBaseFeeScalarConversion)
    })?;
    let l1_base_fee = U256::try_from_be_slice(&data[32..64])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BaseFeeConversion))?;
    let l1_blob_base_fee = U256::try_from_be_slice(&data[64..96])
        .ok_or(OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::BlobBaseFeeConversion))?;
    let operator_fee_scalar = U256::try_from_be_slice(&data[160..164]).ok_or({
        OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::OperatorFeeScalarConversion)
    })?;
    let operator_fee_constant = U256::try_from_be_slice(&data[164..172]).ok_or({
        OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::OperatorFeeConstantConversion)
    })?;

    let mut l1block = L1BlockInfo::default();
    l1block.l1_base_fee = l1_base_fee;
    l1block.l1_base_fee_scalar = l1_base_fee_scalar;
    l1block.l1_blob_base_fee = Some(l1_blob_base_fee);
    l1block.l1_blob_base_fee_scalar = Some(l1_blob_base_fee_scalar);
    l1block.operator_fee_scalar = Some(operator_fee_scalar);
    l1block.operator_fee_constant = Some(operator_fee_constant);

    Ok(l1block)
}

/// An extension trait for [`L1BlockInfo`] that allows us to calculate the L1 cost of a transaction
/// based off of the chain spec's activated hardfork.
pub trait RethL1BlockInfo {
    /// Forwards an L1 transaction calculation to revm and returns the gas cost.
    ///
    /// ### Takes
    /// - `chain_spec`: The chain spec for the node.
    /// - `timestamp`: The timestamp of the current block.
    /// - `input`: The calldata of the transaction.
    /// - `is_deposit`: Whether or not the transaction is a deposit.
    fn l1_tx_data_fee(
        &mut self,
        chain_spec: impl OpHardforks,
        timestamp: u64,
        block: u64,
        input: &[u8],
        is_deposit: bool,
    ) -> Result<U256, BlockExecutionError>;

    /// Computes the data gas cost for an L2 transaction.
    ///
    /// ### Takes
    /// - `chain_spec`: The chain spec for the node.
    /// - `timestamp`: The timestamp of the current block.
    /// - `input`: The calldata of the transaction.
    fn l1_data_gas(
        &self,
        chain_spec: impl OpHardforks,
        timestamp: u64,
        block_number: u64,
        input: &[u8],
    ) -> Result<U256, BlockExecutionError>;
}

impl RethL1BlockInfo for L1BlockInfo {
    fn l1_tx_data_fee(
        &mut self,
        chain_spec: impl OpHardforks,
        timestamp: u64,
        block_number: u64,
        input: &[u8],
        is_deposit: bool,
    ) -> Result<U256, BlockExecutionError> {
        if is_deposit {
            return Ok(U256::ZERO);
        }

        let spec_id = if chain_spec.is_fjord_active_at_timestamp(timestamp) {
            OpSpecId::FJORD
        } else if chain_spec.is_ecotone_active_at_timestamp(timestamp) {
            OpSpecId::ECOTONE
        } else if chain_spec.is_regolith_active_at_timestamp(timestamp) {
            OpSpecId::REGOLITH
        } else if chain_spec.is_bedrock_active_at_block(block_number) {
            OpSpecId::BEDROCK
        } else {
            return Err(
                OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::HardforksNotActive).into()
            );
        };
        Ok(self.calculate_tx_l1_cost(input, spec_id))
    }

    fn l1_data_gas(
        &self,
        chain_spec: impl OpHardforks,
        timestamp: u64,
        block_number: u64,
        input: &[u8],
    ) -> Result<U256, BlockExecutionError> {
        let spec_id = if chain_spec.is_fjord_active_at_timestamp(timestamp) {
            OpSpecId::FJORD
        } else if chain_spec.is_regolith_active_at_timestamp(timestamp) {
            OpSpecId::REGOLITH
        } else if chain_spec.is_bedrock_active_at_block(block_number) {
            OpSpecId::BEDROCK
        } else {
            return Err(
                OpBlockExecutionError::L1BlockInfo(L1BlockInfoError::HardforksNotActive).into()
            );
        };
        Ok(self.data_gas(input, spec_id))
    }
}

/// The Canyon hardfork issues an irregular state transition that force-deploys the create2
/// deployer contract. This is done by directly setting the code of the create2 deployer account
/// prior to executing any transactions on the timestamp activation of the fork.
pub fn ensure_create2_deployer<DB>(
    chain_spec: impl OpHardforks,
    timestamp: u64,
    db: &mut revm_database::State<DB>,
) -> Result<(), DB::Error>
where
    DB: revm::Database,
{
    // If the canyon hardfork is active at the current timestamp, and it was not active at the
    // previous block timestamp (heuristically, block time is not perfectly constant at 2s), and the
    // chain is an optimism chain, then we need to force-deploy the create2 deployer contract.
    if chain_spec.is_canyon_active_at_timestamp(timestamp) &&
        !chain_spec.is_canyon_active_at_timestamp(timestamp.saturating_sub(2))
    {
        trace!(target: "evm", "Forcing create2 deployer contract deployment on Canyon transition");

        // Load the create2 deployer account from the cache.
        let acc = db.load_cache_account(CREATE_2_DEPLOYER_ADDR)?;

        // Update the account info with the create2 deployer codehash and bytecode.
        let mut acc_info = acc.account_info().unwrap_or_default();
        acc_info.code_hash = CREATE_2_DEPLOYER_CODEHASH;
        acc_info.code = Some(Bytecode::new_raw(Bytes::from_static(&CREATE_2_DEPLOYER_BYTECODE)));

        // Convert the cache account back into a revm account and mark it as touched.
        let mut revm_acc: revm::state::Account = acc_info.into();
        revm_acc.mark_touch();

        // Commit the create2 deployer account to the database.
        db.commit(HashMap::from_iter([(CREATE_2_DEPLOYER_ADDR, revm_acc)]));
        return Ok(())
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Block, BlockBody};
    use alloy_eips::eip2718::Decodable2718;
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_forks::OpHardforks;
    use reth_optimism_primitives::OpTransactionSigned;

    #[test]
    fn sanity_l1_block() {
        use alloy_consensus::Header;
        use alloy_primitives::{hex_literal::hex, Bytes};

        let bytes = Bytes::from_static(&hex!("7ef9015aa044bae9d41b8380d781187b426c6fe43df5fb2fb57bd4466ef6a701e1f01e015694deaddeaddeaddeaddeaddeaddeaddeaddead000194420000000000000000000000000000000000001580808408f0d18001b90104015d8eb900000000000000000000000000000000000000000000000000000000008057650000000000000000000000000000000000000000000000000000000063d96d10000000000000000000000000000000000000000000000000000000000009f35273d89754a1e0387b89520d989d3be9c37c1f32495a88faf1ea05c61121ab0d1900000000000000000000000000000000000000000000000000000000000000010000000000000000000000002d679b567db6187c0c8323fa982cfb88b74dbcc7000000000000000000000000000000000000000000000000000000000000083400000000000000000000000000000000000000000000000000000000000f4240"));
        let l1_info_tx = OpTransactionSigned::decode_2718(&mut bytes.as_ref()).unwrap();
        let mock_block = Block {
            header: Header::default(),
            body: BlockBody { transactions: vec![l1_info_tx], ..Default::default() },
        };

        let l1_info: L1BlockInfo = extract_l1_info(&mock_block.body).unwrap();
        assert_eq!(l1_info.l1_base_fee, U256::from(652_114));
        assert_eq!(l1_info.l1_fee_overhead, Some(U256::from(2100)));
        assert_eq!(l1_info.l1_base_fee_scalar, U256::from(1_000_000));
        assert_eq!(l1_info.l1_blob_base_fee, None);
        assert_eq!(l1_info.l1_blob_base_fee_scalar, None);
    }

    #[test]
    fn sanity_l1_block_ecotone() {
        // rig

        // OP mainnet ecotone block 118024092
        // <https://optimistic.etherscan.io/block/118024092>
        const TIMESTAMP: u64 = 1711603765;
        assert!(OP_MAINNET.is_ecotone_active_at_timestamp(TIMESTAMP));

        // First transaction in OP mainnet block 118024092
        //
        // https://optimistic.etherscan.io/getRawTx?tx=0x88501da5d5ca990347c2193be90a07037af1e3820bb40774c8154871c7669150
        const TX: [u8; 251] = hex!("7ef8f8a0a539eb753df3b13b7e386e147d45822b67cb908c9ddc5618e3dbaa22ed00850b94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e2000000558000c5fc50000000000000000000000006605a89f00000000012a10d90000000000000000000000000000000000000000000000000000000af39ac3270000000000000000000000000000000000000000000000000000000d5ea528d24e582fa68786f080069bdbfe06a43f8e67bfd31b8e4d8a8837ba41da9a82a54a0000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985");

        let tx = OpTransactionSigned::decode_2718(&mut TX.as_slice()).unwrap();
        let block: Block<OpTransactionSigned> = Block {
            body: BlockBody { transactions: vec![tx], ..Default::default() },
            ..Default::default()
        };

        // expected l1 block info
        let expected_l1_base_fee = U256::from_be_bytes(hex!(
            "0000000000000000000000000000000000000000000000000000000af39ac327" // 47036678951
        ));
        let expected_l1_base_fee_scalar = U256::from(1368);
        let expected_l1_blob_base_fee = U256::from_be_bytes(hex!(
            "0000000000000000000000000000000000000000000000000000000d5ea528d2" // 57422457042
        ));
        let expected_l1_blob_base_fee_scalar = U256::from(810949);

        // test

        let l1_block_info: L1BlockInfo = extract_l1_info(&block.body).unwrap();

        assert_eq!(l1_block_info.l1_base_fee, expected_l1_base_fee);
        assert_eq!(l1_block_info.l1_base_fee_scalar, expected_l1_base_fee_scalar);
        assert_eq!(l1_block_info.l1_blob_base_fee, Some(expected_l1_blob_base_fee));
        assert_eq!(l1_block_info.l1_blob_base_fee_scalar, Some(expected_l1_blob_base_fee_scalar));
    }

    #[test]
    fn parse_l1_info_fjord() {
        // rig

        // L1 block info for OP mainnet block 124665056 (stored in input of tx at index 0)
        //
        // https://optimistic.etherscan.io/tx/0x312e290cf36df704a2217b015d6455396830b0ce678b860ebfcc30f41403d7b1
        const DATA: &[u8] = &hex!("440a5e200000146b000f79c500000000000000040000000066d052e700000000013ad8a3000000000000000000000000000000000000000000000000000000003ef1278700000000000000000000000000000000000000000000000000000000000000012fdf87b89884a61e74b322bbcf60386f543bfae7827725efaaf0ab1de2294a590000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985");

        // expected l1 block info verified against expected l1 fee for tx. l1 tx fee listed on OP
        // mainnet block scanner
        //
        // https://github.com/bluealloy/revm/blob/fa5650ee8a4d802f4f3557014dd157adfb074460/crates/revm/src/optimism/l1block.rs#L414-L443
        let l1_base_fee = U256::from(1055991687);
        let l1_base_fee_scalar = U256::from(5227);
        let l1_blob_base_fee = Some(U256::from(1));
        let l1_blob_base_fee_scalar = Some(U256::from(1014213));

        // test

        let l1_block_info = parse_l1_info(DATA).unwrap();

        assert_eq!(l1_block_info.l1_base_fee, l1_base_fee);
        assert_eq!(l1_block_info.l1_base_fee_scalar, l1_base_fee_scalar);
        assert_eq!(l1_block_info.l1_blob_base_fee, l1_blob_base_fee);
        assert_eq!(l1_block_info.l1_blob_base_fee_scalar, l1_blob_base_fee_scalar);
    }

    #[test]
    fn parse_l1_info_isthmus() {
        // rig

        // L1 block info from a devnet with Isthmus activated
        const DATA: &[u8] = &hex!("098999be00000558000c5fc500000000000000030000000067a9f765000000000000002900000000000000000000000000000000000000000000000000000000006a6d09000000000000000000000000000000000000000000000000000000000000000172fcc8e8886636bdbe96ba0e4baab67ea7e7811633f52b52e8cf7a5123213b6f000000000000000000000000d3f2c5afb2d76f5579f326b0cd7da5f5a4126c3500004e2000000000000001f4");

        // expected l1 block info verified against expected l1 fee and operator fee for tx.
        let l1_base_fee = U256::from(6974729);
        let l1_base_fee_scalar = U256::from(1368);
        let l1_blob_base_fee = Some(U256::from(1));
        let l1_blob_base_fee_scalar = Some(U256::from(810949));
        let operator_fee_scalar = Some(U256::from(20000));
        let operator_fee_constant = Some(U256::from(500));

        // test

        let l1_block_info = parse_l1_info(DATA).unwrap();

        assert_eq!(l1_block_info.l1_base_fee, l1_base_fee);
        assert_eq!(l1_block_info.l1_base_fee_scalar, l1_base_fee_scalar);
        assert_eq!(l1_block_info.l1_blob_base_fee, l1_blob_base_fee);
        assert_eq!(l1_block_info.l1_blob_base_fee_scalar, l1_blob_base_fee_scalar);
        assert_eq!(l1_block_info.operator_fee_scalar, operator_fee_scalar);
        assert_eq!(l1_block_info.operator_fee_constant, operator_fee_constant);
    }
}
