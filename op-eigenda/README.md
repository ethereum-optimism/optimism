OP x EigenDA
==============

This package defines a CLI utility for looking up on EigenDA a rollup blob referenced in an OpStack calldata write.

<!-- TODO: Update the documentation -->

<!--
`op-batcher` and `op-node` need to be updated to use the new `BlobInfo` instead:

in `calldata_source.go` `DataFromEVMTransactions`

 out = append(out, tx.Data())

becomes

    frameRef := celestia.FrameRef{}
    frameRef.UnmarshalBinary(tx.Data())
    if err != nil {
     log.Warn("unable to decode frame reference", "index", j, "err", err)
     return nil, err
    }
    log.Info("requesting data from celestia", "namespace", hex.EncodeToString(daCfg.Namespace), "height", frameRef.BlockHeight)
    blob, err := daCfg.Client.Blob.Get(context.Background(), frameRef.BlockHeight, daCfg.Namespace, frameRef.TxCommitment)
    if err != nil {
     return nil, NewResetError(fmt.Errorf("failed to resolve frame from celestia: %w", err))
    }
    out = append(out, blob.Data)

in `txmgr/txmgr.go`:

  tx, err := m.craftTx(ctx, candidate)

becomes

  dataBlob, err := blob.NewBlobV0(m.namespace.Bytes(), candidate.TxData)
  com, err := blob.CreateCommitment(dataBlob)
  if err != nil {
   m.l.Warn("unable to create blob commitment to celestia", "err", err)
   return nil, err
  }
  height, err := m.daClient.Blob.Submit(ctx, []*blob.Blob{dataBlob})
  if err != nil {
   m.l.Warn("unable to publish tx to celestia", "err", err)
   return nil, err
  }
  if height == 0 {
   m.l.Warn("unexpected response from celestia got", "height", height)
   return nil, errors.New("unexpected response code")
  }
  frameRef := celestia.FrameRef{
   BlockHeight: height,
   TxCommitment: com,
  }
  frameRefData, _ := frameRef.MarshalBinary()
  candidate = TxCandidate{TxData: frameRefData, To: candidate.To, GasLimit: candidate.GasLimit} -->
