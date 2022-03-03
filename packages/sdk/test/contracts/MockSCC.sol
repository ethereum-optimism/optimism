pragma solidity ^0.8.9;

contract MockSCC {
    event StateBatchAppended(
        uint256 indexed _batchIndex,
        bytes32 _batchRoot,
        uint256 _batchSize,
        uint256 _prevTotalElements,
        bytes _extraData
    );

    struct StateBatchAppendedArgs {
        uint256 batchIndex;
        bytes32 batchRoot;
        uint256 batchSize;
        uint256 prevTotalElements;
        bytes extraData;
    }

    // Window in seconds, will resolve to 100 blocks.
    uint256 public FRAUD_PROOF_WINDOW = 1500;
    uint256 public batches = 0;
    StateBatchAppendedArgs public sbaParams;

    function getTotalBatches() public view returns (uint256) {
        return batches;
    }

    function setSBAParams(
        StateBatchAppendedArgs memory _args
    ) public {
        sbaParams = _args;
    }

    function appendStateBatch(
        bytes32[] memory _roots,
        uint256 _shouldStartAtIndex
    ) public {
        batches++;
        emit StateBatchAppended(
            sbaParams.batchIndex,
            sbaParams.batchRoot,
            sbaParams.batchSize,
            sbaParams.prevTotalElements,
            sbaParams.extraData
        );
    }
}
