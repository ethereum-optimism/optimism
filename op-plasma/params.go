package plasma

// MaxInputSize ensures the canonical chain cannot include input batches too large to
// challenge in the Data Availability Challenge contract. Value in number of bytes.
// This value can only be changed in a hard fork.
const MaxInputSize = 130672

// TxDataVersion1 is the version number for batcher transactions containing
// plasma commitments. It should not collide with DerivationVersion which is still
// used downstream when parsing the frames.
const TxDataVersion1 = 1
