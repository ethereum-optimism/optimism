package eigenda

type Cert struct {
	BatchHeaderHash      []byte   `rlp:"batchHeaderHash"`
	BlobIndex            uint32   `rlp:"blobIndex"`
	ReferenceBlockNumber uint32   `rlp:"referenceBlockNumber"`
	QuorumIDs            []uint32 `rlp:"quorumIDs"`
}
