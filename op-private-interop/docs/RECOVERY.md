# Private projection recovery

The private LightCL retains normal L1 confirmation settings. Its claimed follow endpoint
provides a private checkpoint and the projection's fully scanned local-safe frontier.
After fallback or replacement, the LightCL pauses sequencing and uses the ordinary
attributes handler to execute canonical deposit-only inputs against private state. It
checks the projection's timestamp, L1 origin and sequence number; projection hashes
identify inputs and are never used as private forkchoice hashes.

Local-safe execution, cross-safety and finality remain separate. The adapter maps the
projection's safety frontiers through the authenticated private ancestry. A projection
reorg revokes affected private checkpoints; finalized private history cannot be revoked.
For a claim whose carrier survives but whose suffix is replaced, the parent hash in
the accepted private terminal commitment authenticates the surviving prefix by ancestry. The
supernode's durable deny list and retained denied headers reconstruct that information
after restart. Missing headers or private state stop recovery.

The supernode keeps each claim's observed branch tip and recomputes its surviving
prefix from canonical history at every completed scan. It does not maintain separate
revocation, completion or restoration flags. One sparse read of the durable deny list
also caps recovery before any still-canonical denied block, including a denied claim
carrier or an ancestor of a later claim, while engine rewind is pending. Denials added
after that read become visible on the next poll or reset; this RPC is a polled snapshot.
A temporary safety-label retreat preserves the scan journal so the same branch can
advance again.
LightCL uses the engine controller's local-safe head as its execution cursor and only
tracks the corresponding public input and any pending build.

The replay interval is `(surviving private parent, recovery frontier]`. LightCL
walks backward from the committed terminal parent to resolve the surviving private
parent locally. An offset from an older checkpoint alone cannot identify the private
branch. The invalidated terminal header is unnecessary: the parent hash is itself
part of the operator's attestation, under the same trust policy as the rest of the
claim. A future proof verifier must bind that parent to the proven private execution.
Offsets are implicit in the endpoint heights, so no separate offset field is needed.

The experimental claimed-follow RPC prefix now carries `parent` (the committed private
terminal parent's block ID) and `last` (the surviving projection reference), replacing
`terminal` and `terminal_parent`. Run matching supernode and LightCL builds when upgrading;
the batch commitment format is unchanged.

Private batchers require `--private-interop.public-projection-rollup-rpc` alongside the
projection execution RPC. The publication cursor uses the projection's local-safe head
and waits until private canonical execution has reconciled it. A surviving claim carrier
still reserves its original range, so recovery executes actual canonical replacements
through that range before publication resumes. If canonical derivation drops the
remaining span, those positions wait for ordinary sequencing-window expiry. Range
publication currently waits for the previous projection parent. Catch-up therefore requires enough blocks per range to
outpace L1 inclusion plus follow polling; undersized ranges can remain at the expiry
frontier even when reconciliation is correct. The outage test retains an eight-block
catch-up limit and uses the devstack/op-up default of twelve-block ranges with its
accelerated L1.

Recovery consumes replacements produced by canonical derivation and cross-safety checks.
A reverted ClaimRegistry call alone does not invalidate its block or suppress separate
replay calls; adding a new proof verifier requires a consensus rule or atomic replay
authorization that makes invalid proof handling effective. This adapter does not provide
that rule.
