//! ABI shared with the experimental Solidity router.
use alloy_sol_types::sol;
sol! {
    #[derive(Debug, Default, PartialEq, Eq)]
    struct Identifier { address origin; uint256 blockNumber; uint256 logIndex; uint256 timestamp; uint256 chainId; }
    #[derive(Debug, Default, PartialEq, Eq)]
    struct AtomicResultWitness { Identifier identifier; bool success; bytes returnData; }
    #[derive(Debug, Default, PartialEq, Eq)]
    struct AtomicRemoteCall { Identifier identifier; uint256 sequence; address sender; address target; bytes data; }
    #[derive(Debug, Default, PartialEq, Eq)]
    struct AtomicCallback { uint256 waitingSequence; AtomicRemoteCall call; bool success; }
    #[derive(Debug)]
    struct AtomicWitnessRequest { uint256 sequence; uint256 chainId; address target; address sender; bytes data; }
    #[derive(Debug)]
    struct AtomicStreamCursor { uint256 index; bytes previousResult; }
    function executeRootWithGas(uint256 nonce, address target, bytes data, AtomicResultWitness[] witnesses, uint64 applicationGas) returns(bytes);
    function executeRemoteWithGas(bytes32 bundleId, AtomicRemoteCall[] calls, AtomicResultWitness[] witnesses, Identifier rootCompletion, uint64 applicationGas, uint16 maxCalls) returns(bytes[]);
    function executeRootNested(uint256 nonce, address target, bytes data, AtomicResultWitness[] witnesses, AtomicCallback[] callbacks, uint64 applicationGas) returns(bytes);
    function executeRemoteNested(bytes32 bundleId, AtomicRemoteCall[] calls, AtomicResultWitness[] witnesses, AtomicCallback[] callbacks, Identifier rootCompletion, uint64 applicationGas, uint16 maxCalls) returns(bytes[]);
    function callbackAt(AtomicWitnessRequest request, uint256 index) external view returns(bool found, AtomicRemoteCall item);
    function callFinished(uint256 sourceChain, uint256 sequence, bool success, bytes result) external view;
    function callbackStatus() external view returns(bool failed, uint256 total);
    function witnessAt(AtomicWitnessRequest request) external view returns (bool found, AtomicResultWitness witness);
    function remoteCallAt(AtomicStreamCursor cursor) external view returns (bool found, AtomicRemoteCall item);
    function witnessCount() external view returns(uint256);
    function completionIdentifier() external view returns(Identifier);
    function validateMessage(Identifier id, bytes32 msgHash);
    event CallRequested(bytes32 indexed callId, bytes32 requestHash);
    event CallResult(bytes32 indexed callId, bytes32 resultHash);
    event BundleCompleted(bytes32 indexed bundleId);
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);
}
