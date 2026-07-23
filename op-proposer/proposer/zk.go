package proposer

import "encoding/binary"

// ZKDisputeGameType is the dispute game type for the super-root ZK dispute
// game. It mirrors op-challenger/game/types.ZKDisputeGameType; op-proposer
// follows its convention of raw uint32 game types (see postInteropGameTypes).
const ZKDisputeGameType uint32 = 10

// zkExtraData encodes the extraData for a ZKDisputeGame creation:
// a 4-byte big-endian parent game index followed by the super root proof
// (the marshaled super root pre-image). See the extraData layout documented
// in ZKDisputeGame.sol and the CWIA getters parentIndex()/l2SequenceNumber().
func zkExtraData(parentIndex uint32, superRootProof []byte) []byte {
	extraData := make([]byte, 4+len(superRootProof))
	binary.BigEndian.PutUint32(extraData[:4], parentIndex)
	copy(extraData[4:], superRootProof)
	return extraData
}
