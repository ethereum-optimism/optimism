// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

using LPPMetadataLib for LPPMetaData global;

/// @notice Packed LPP metadata.
/// ┌─────────────┬────────────────────────────────────────────┐
/// │ Bit Offsets │                Description                 │
/// ├─────────────┼────────────────────────────────────────────┤
/// │ [0, 64)     │ Timestamp (Finalized - All data available) │
/// │ [64, 96)    │ Part Offset                                │
/// │ [96, 128)   │ Claimed Size                               │
/// │ [128, 160)  │ Blocks Processed (Inclusive of Padding)    │
/// │ [160, 192)  │ Bytes Processed (Non-inclusive of Padding) │
/// │ [192, 256)  │ Countered                                  │
/// └─────────────┴────────────────────────────────────────────┘
type LPPMetaData is bytes32;

/// @notice LPP metadata UDT extension functions.
library LPPMetadataLib {
    function setTimestamp(LPPMetaData _self, uint64 _timestamp) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(shl(192, _timestamp), and(_self, not(shl(192, 0xFFFFFFFFFFFFFFFF))))
        }
    }

    function setPartOffset(LPPMetaData _self, uint32 _partOffset) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(shl(160, _partOffset), and(_self, not(shl(160, 0xFFFFFFFF))))
        }
    }

    function setClaimedSize(LPPMetaData _self, uint32 _claimedSize) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(shl(128, _claimedSize), and(_self, not(shl(128, 0xFFFFFFFF))))
        }
    }

    function setBlocksProcessed(LPPMetaData _self, uint32 _blocksProcessed) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(shl(96, _blocksProcessed), and(_self, not(shl(96, 0xFFFFFFFF))))
        }
    }

    function setBytesProcessed(LPPMetaData _self, uint32 _bytesProcessed) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(shl(64, _bytesProcessed), and(_self, not(shl(64, 0xFFFFFFFF))))
        }
    }

    function setCountered(LPPMetaData _self, bool _countered) internal pure returns (LPPMetaData self_) {
        assembly {
            self_ := or(_countered, and(_self, not(0xFFFFFFFFFFFFFFFF)))
        }
    }

    function timestamp(LPPMetaData _self) internal pure returns (uint64 timestamp_) {
        assembly {
            timestamp_ := shr(192, _self)
        }
    }

    function partOffset(LPPMetaData _self) internal pure returns (uint64 partOffset_) {
        assembly {
            partOffset_ := and(shr(160, _self), 0xFFFFFFFF)
        }
    }

    function claimedSize(LPPMetaData _self) internal pure returns (uint32 claimedSize_) {
        assembly {
            claimedSize_ := and(shr(128, _self), 0xFFFFFFFF)
        }
    }

    function blocksProcessed(LPPMetaData _self) internal pure returns (uint32 blocksProcessed_) {
        assembly {
            blocksProcessed_ := and(shr(96, _self), 0xFFFFFFFF)
        }
    }

    function bytesProcessed(LPPMetaData _self) internal pure returns (uint32 bytesProcessed_) {
        assembly {
            bytesProcessed_ := and(shr(64, _self), 0xFFFFFFFF)
        }
    }

    function countered(LPPMetaData _self) internal pure returns (bool countered_) {
        assembly {
            countered_ := and(_self, 0xFFFFFFFFFFFFFFFF)
        }
    }
}
