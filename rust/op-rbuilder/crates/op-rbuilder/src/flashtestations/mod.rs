use alloy_sol_types::sol;

// https://github.com/flashbots/flashtestations/commit/7cc7f68492fe672a823dd2dead649793aac1f216
sol!(
    #[sol(rpc, abi)]
    #[derive(Debug)]
    interface IFlashtestationRegistry {
        function registerTEEService(bytes calldata rawQuote, bytes calldata extendedRegistrationData) external;

        function permitRegisterTEEService(
            bytes calldata rawQuote,
            bytes calldata extendedRegistrationData,
            uint256 nonce,
            uint256 deadline,
            bytes calldata signature
        ) external payable;

        function computeStructHash(
            bytes calldata rawQuote,
            bytes calldata extendedRegistrationData,
            uint256 nonce,
            uint256 deadline
        ) external pure returns (bytes32);

        function hashTypedDataV4(bytes32 structHash) external view returns (bytes32);

        function getRegistrationStatus(address teeAddress) external view returns (bool isValid, bytes32 quoteHash);

        /// @notice Emitted when a TEE service is registered
        /// @param teeAddress The address of the TEE service
        /// @param rawQuote The raw quote from the TEE device
        /// @param alreadyExists Whether the TEE service is already registered
        event TEEServiceRegistered(address indexed teeAddress, bytes rawQuote, bool alreadyExists);

        /// @notice Emitted when the attestation contract is the 0x0 address
        error InvalidAttestationContract();
        /// @notice Emitted when the signature is expired because the deadline has passed
        error ExpiredSignature(uint256 deadline);
        /// @notice Emitted when the quote is invalid according to the Automata DCAP Attestation contract
        error InvalidQuote(bytes output);
        /// @notice Emitted when the report data length is too short
        error InvalidReportDataLength(uint256 length);
        /// @notice Emitted when the registration data hash does not match the expected hash
        error InvalidRegistrationDataHash(bytes32 expected, bytes32 received);
        /// @notice Emitted when the byte size is exceeded
        error ByteSizeExceeded(uint256 size);
        /// @notice Emitted when the TEE service is already registered when registering
        error TEEServiceAlreadyRegistered(address teeAddress);
        /// @notice Emitted when the signer doesn't match the TEE address
        error SignerMustMatchTEEAddress(address signer, address teeAddress);
        /// @notice Emitted when the TEE service is not registered
        error TEEServiceNotRegistered(address teeAddress);
        /// @notice Emitted when the TEE service is already invalid when trying to invalidate a TEE registration
        error TEEServiceAlreadyInvalid(address teeAddress);
        /// @notice Emitted when the TEE service is still valid when trying to invalidate a TEE registration
        error TEEIsStillValid(address teeAddress);
        /// @notice Emitted when the nonce is invalid when verifying a signature
        error InvalidNonce(uint256 expected, uint256 provided);
    }

    #[sol(rpc, abi)]
    #[derive(Debug)]
    interface IBlockBuilderPolicy {
        function verifyBlockBuilderProof(uint8 version, bytes32 blockContentHash) external;

        function permitVerifyBlockBuilderProof(
            uint8 version,
            bytes32 blockContentHash,
            uint256 nonce,
            bytes calldata eip712Sig
        ) external;

        function computeStructHash(uint8 version, bytes32 blockContentHash, uint256 nonce)
            external
            pure
            returns (bytes32);

        function getHashedTypeDataV4(bytes32 structHash) external view returns (bytes32);

        /// @notice Emitted when a block builder proof is successfully verified
        /// @param caller The address that called the verification function (TEE address)
        /// @param workloadId The workload identifier of the TEE
        /// @param version The flashtestation protocol version used
        /// @param blockContentHash The hash of the block content
        /// @param commitHash The git commit hash associated with the workload
        event BlockBuilderProofVerified(
            address caller, bytes32 workloadId, uint8 version, bytes32 blockContentHash, string commitHash
        );

        /// @notice Emitted when the registry is the 0x0 address
        error InvalidRegistry();
        /// @notice Emitted when a workload to be added is already in the policy
        error WorkloadAlreadyInPolicy();
        /// @notice Emitted when a workload to be removed is not in the policy
        error WorkloadNotInPolicy();
        /// @notice Emitted when the address is not in the approvedWorkloads mapping
        error UnauthorizedBlockBuilder(address caller);
        /// @notice Emitted when the nonce is invalid
        error InvalidNonce(uint256 expected, uint256 provided);
        /// @notice Emitted when the commit hash is empty
        error EmptyCommitHash();
        /// @notice Emitted when the source locators array is empty
        error EmptySourceLocators();
    }

    interface IERC20Permit {
        function nonces(address owner) external view returns (uint256);
    }

    struct BlockData {
        bytes32 parentHash;
        uint256 blockNumber;
        uint256 timestamp;
        bytes32[] transactionHashes;
    }

    type WorkloadId is bytes32;
);

pub mod args;
pub mod attestation;
pub mod builder_tx;
pub mod service;
pub mod tx_manager;
