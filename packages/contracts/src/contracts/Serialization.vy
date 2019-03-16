### BEGIN TRANSACTION DECODING SECION ###

# Note: TX Encoding Lengths
#
# The MAX_TRANSFERS should be a tunable constant, but vyper doesn't 
# support 'bytes[constant var]' so it has to be hardcoded.  The formula
# for calculating the encoding size is:
#     TX_BLOCK_DECODE_LEN + 
#     TX_NUM_TRANSFERS_LEN +
#     MAX_TRANSFERS * TRANSFER_LEN 
# Currently we take MAX_TRANSFERS = 4, so the max TX encoding bytes is:
# 4 + 1 + 4 * 68 = 277

@public
def getLeafHash(transactionEncoding: bytes[277]) -> bytes32:
    return sha3(transactionEncoding)

TX_BLOCKNUM_START: constant(int128) = 0
TX_BLOCKNUM_LEN: constant(int128) = 4
@public
def decodeBlockNumber(transactionEncoding: bytes[277]) -> uint256:
    bn: bytes[32] = slice(transactionEncoding,
            start = TX_BLOCKNUM_START,
            len = TX_BLOCKNUM_LEN)
    return convert(bn, uint256)

TX_NUM_TRANSFERS_START: constant(int128) = 4
TX_NUM_TRANSFERS_LEN: constant(int128) = 1
@public
def decodeNumTransfers(transactionEncoding: bytes[277]) -> uint256:
    num: bytes[2] = slice(transactionEncoding,
            start = TX_NUM_TRANSFERS_START,
            len = TX_NUM_TRANSFERS_LEN)
    return convert(num, uint256)

FIRST_TR_START: constant(int128) = 5
TR_LEN: constant(int128) = 68
@public
def decodeIthTransfer(
    index: int128,
    transactionEncoding: bytes[277]
) -> bytes[68]:
    transfer: bytes[68] = slice(transactionEncoding,
        start = TR_LEN * index + FIRST_TR_START,
        len = TR_LEN
    )
    return transfer

### BEGIN TRANSFER DECODING SECTION ###

@public
def bytes20ToAddress(addr: bytes[20]) -> address:
    padded: bytes[52] = concat(EMPTY_BYTES32, addr)
    return convert(convert(slice(padded, start=20, len=32), bytes32), address)

SENDER_START: constant(int128) = 0
SENDER_LEN: constant(int128) = 20
@public
def decodeSender(
    transferEncoding: bytes[68]
) -> address:
    addr: bytes[20] = slice(transferEncoding,
        start = SENDER_START,
        len = SENDER_LEN)
    return self.bytes20ToAddress(addr)

RECIPIENT_START: constant(int128) = 20
RECIPIENT_LEN: constant(int128) = 20
@public
def decodeRecipient(
    transferEncoding: bytes[68]
) -> address:
    addr: bytes[20] = slice(transferEncoding,
        start = RECIPIENT_START,
        len = RECIPIENT_LEN)
    return self.bytes20ToAddress(addr)

TR_TOKEN_START: constant(int128) = 40
TR_TOKEN_LEN: constant(int128) = 4
@public
def decodeTokenTypeBytes(
    transferEncoding: bytes[68]
) -> bytes[4]:
    tokenType: bytes[4] = slice(transferEncoding, 
        start = TR_TOKEN_START,
        len = TR_TOKEN_LEN)
    return tokenType

@public
def decodeTokenType(
    transferEncoding: bytes[68]
) -> uint256:
    return convert(
        self.decodeTokenTypeBytes(transferEncoding), 
        uint256
    )

@public
def getTypedFromTokenAndUntyped(tokenType: uint256, coinID: uint256) -> uint256:
    return coinID + tokenType * (256**12)

TR_UNTYPEDSTART_START: constant(int128) = 44
TR_UNTYPEDSTART_LEN: constant(int128) = 12
TR_UNTYPEDEND_START: constant(int128) = 56
TR_UNTYPEDEND_LEN: constant(int128) = 12
@public
def decodeTypedTransferRange(
    transferEncoding: bytes[68]
) -> (uint256, uint256): # start, end
    tokenType: bytes[4] = self.decodeTokenTypeBytes(transferEncoding)
    untypedStart: bytes[12] = slice(transferEncoding,
        start = TR_UNTYPEDSTART_START,
        len = TR_UNTYPEDSTART_LEN)
    untypedEnd: bytes[12] = slice(transferEncoding,
        start = TR_UNTYPEDEND_START,
        len = TR_UNTYPEDEND_LEN)
    return (
        convert(concat(tokenType, untypedStart), uint256),
        convert(concat(tokenType, untypedEnd), uint256)
    )

### BEGIN TRANSFERPROOF DECODING SECTION ###

# Note on TransferProofEncoding size:
# It will always really be at most 
# PARSED_SUM_LEN + LEAF_INDEX_LEN + ADDRESS_LEN + PROOF_COUNT_LEN + MAX_TREE_DEPTH * TREENODE_LEN
# = 16 + 16 + 20 + 1 + 8 * 48 = 437
# but because of dumb type casting in vyper, it thinks it *might* 
# be larger because we slice the TX encoding to get it.  So it has to be
# TRANSFERPROOF_COUNT_LEN + 437 * MAX_TRANSFERS = 1 + 1744 * 4 = 1749

TREENODE_LEN: constant(int128) = 48

PARSEDSUM_START: constant(int128) = 0
PARSEDSUM_LEN: constant(int128) = 16
@public
def decodeParsedSumBytes(
    transferProofEncoding: bytes[1749] 
) -> bytes[16]:
    parsedSum: bytes[16] = slice(transferProofEncoding,
        start = PARSEDSUM_START,
        len = PARSEDSUM_LEN)
    return parsedSum

LEAFINDEX_START: constant(int128) = 16
LEAFINDEX_LEN: constant(int128) = 16
@public
def decodeLeafIndex(
    transferProofEncoding: bytes[1749]
) -> int128:
    leafIndex: bytes[16] = slice(transferProofEncoding,
        start = LEAFINDEX_START,
        len = PARSEDSUM_LEN)
    return convert(leafIndex, int128)

SIG_START:constant(int128) = 32
SIGV_OFFSET: constant(int128) = 0
SIGV_LEN: constant(int128) = 1
SIGR_OFFSET: constant(int128) = 1
SIGR_LEN: constant(int128) = 32
SIGS_OFFSET: constant(int128) = 33
SIGS_LEN: constant(int128) = 32
@public
def decodeSignature(
    transferProofEncoding: bytes[1749]
) -> (
    uint256, # v
    uint256, # r
    uint256 # s
):
    sig: bytes[65] = slice(transferProofEncoding,
        start = SIG_START,
        len = SIGV_LEN + SIGR_LEN + SIGS_LEN
    )
    sigV: bytes[1] = slice(sig,
        start = SIGV_OFFSET,
        len = SIGV_LEN)
    sigR: bytes[32] = slice(sig,
        start = SIGR_OFFSET,
        len = SIGR_LEN)
    sigS: bytes[32] = slice(sig,
        start = SIGS_OFFSET,
        len = SIGS_LEN)
    return (
        convert(sigV, uint256),
        convert(sigR, uint256),
        convert(sigS, uint256)
    )

NUMPROOFNODES_START: constant(int128) = 97
NUMPROOFNODES_LEN: constant(int128) = 1
@public
def decodeNumInclusionProofNodesFromTRProof(transferProof: bytes[1749]) -> int128:
    numNodes: bytes[1] = slice(
        transferProof,
        start = NUMPROOFNODES_START,
        len = NUMPROOFNODES_LEN
    )
    return convert(numNodes, int128)

INCLUSIONPROOF_START: constant(int128) = 98
@public
def decodeIthInclusionProofNode(
    index: int128,
    transferProofEncoding: bytes[1749]
) -> bytes[48]: # = MAX_TREE_DEPTH * TREENODE_LEN = 384 is what it should be but because of variable in slice vyper won't let us say that :(
    proofNode: bytes[48] = slice(transferProofEncoding, 
        start = index * TREENODE_LEN + INCLUSIONPROOF_START,
        len =  TREENODE_LEN)
    return proofNode

### BEGIN TRANSACTION PROOF DECODING SECTION ###

# The smart contract assumes the number of nodes in every TRProof are equal.
FIRST_TRANSFERPROOF_START: constant(int128) = 1
@public
def decodeNumInclusionProofNodesFromTXProof(transactionProof: bytes[1749]) -> int128:
    firstTransferProof: bytes[1749] = slice(
        transactionProof,
        start = FIRST_TRANSFERPROOF_START,
        len = NUMPROOFNODES_START + 1 # + 1 so we include the numNodes
    )
    return self.decodeNumInclusionProofNodesFromTRProof(firstTransferProof)


NUMTRPROOFS_START: constant(int128) = 0
NUMTRPROOFS_LEN: constant(int128) = 1
@public
def decodeNumTransactionProofs(
    transactionProofEncoding: bytes[1749]
) -> int128:
    numInclusionProofs: bytes[1] = slice(
        transactionProofEncoding,
        start = NUMTRPROOFS_START,
        len = NUMTRPROOFS_LEN
    )
    return convert(numInclusionProofs, int128)

@public
def decodeIthTransferProofWithNumNodes(
    index: int128,
    numInclusionProofNodes: int128,
    transactionProofEncoding: bytes[1749]
) -> bytes[1749]:
    transactionProofLen: int128 = (
        #PARSEDSUM_LEN + #16
        #LEAFINDEX_LEN + #16
        #SIGS_LEN + SIGV_LEN + SIGR_LEN + # 65
        #NUMPROOFNODES_LEN + #1
        98 + TREENODE_LEN * numInclusionProofNodes
    )
    transferProof: bytes[1749] = slice(
        transactionProofEncoding,
        start = index * transactionProofLen + FIRST_TRANSFERPROOF_START,
        len = transactionProofLen
    )
    return transferProof
