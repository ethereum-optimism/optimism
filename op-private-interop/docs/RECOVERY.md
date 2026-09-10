# Private projection recovery

The private LightCL retains normal L1 confirmation settings. Its claimed follow endpoint
provides a private checkpoint and the projection's fully scanned local-safe frontier.
After fallback or replacement, the LightCL pauses sequencing and uses the ordinary
attributes handler to execute canonical deposit-only inputs against private state. It
checks the projection's timestamp, L1 origin and sequence number; projection hashes
identify inputs and are never used as private forkchoice hashes.

## Why a rewind works

An execution block identifies its parent and the resulting state root. Selecting
an older private block as the execution head selects that block's private state.
It removes the later suffix from the canonical chain; it does not try to undo
each application write individually. This requires the private execution client
to retain the necessary block and state data.

For example, suppose private block 40 is the surviving checkpoint and public
blocks 41–46 have become deposit-only fallback blocks:

1. Pause private sequencing and cancel any unfinished build.
2. Authenticate private block 40 and its ancestry against the safety labels.
3. Set private forkchoice to that checkpoint using the ordinary engine reset.
4. For each public position 41–46, fetch its canonical L1 origin, timestamp and
   sequence number. Build the corresponding private block from its **private**
   parent with the ordinary L1 attributes and deposit builder, without txpool
   transactions.
5. Resume sequencing when the required interval is complete. If a surviving
   prefix belongs to a larger rejected publication range, wait for that whole
   reserved range before resuming.

The same deposit can execute again on the replacement branch without being
counted twice in canonical private state: execution starts from the selected
parent's state. Ordinary private deposits remain enabled during recovery.
Private and public replacement blocks need not have the same hash or state root.

## Canonical recovery intervals

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

## The supernode contract

The private follow route is `<chain route>/claimed`:

- `optimism_syncStatus` carries normal private safety labels plus the
  `private_recovery` plan: an authenticated private anchor, a public target,
  public safe/finalized positions and, when needed, an attested surviving prefix.
- `optimism_recoveryBlock(number, target)` returns the public block reference
  for a recovery position. The target includes a hash, so requests are tied to a
  canonical frontier. The server rejects revoked plans, changed targets and
  intervals containing sequencer transactions.

The recovery field is a plan, not a one-shot command. LightCL checks whether its
current execution and previously applied public references still match it.
An ordinary public follow source that never supplies this field continues using
the normal follow path. Once private recovery has been enabled, silently losing
the field pauses progress rather than changing synchronization modes.

## Reorgs and safety

LightCL checks follow snapshots against its own L1 connection. Its ordinary
sequencer also detects orphaned L1 origins and requests an engine reset. The
supernode is still needed to report public claim loss or interop invalidation:
those cannot be inferred just from the private blocks' L1 origins.

A lower accepted checkpoint can cause a conservative rewind even if the private
suffix's L1 inputs have not changed. Keep this behavior: a temporarily lower
public safety frontier and a pending claim-carrier invalidation can expose the
same recovery plan. `target == anchor` with no prefix is not evidence that the
unsafe suffix is valid.

Recovery authenticates private ancestry, preserves finalized history, and stops
on unavailable data or inconsistent inputs. An RPC failure is not permission to
invent an origin schedule or continue sequencing. The stock sequencer's recovery
flag alone is insufficient: its current L1-origin choice may differ from the
canonical public schedule being recovered.

## Regression coverage

- `op-node/rollup/driver/follow_recovery*_test.go`: canonical attributes, branch
  authentication, interval completion, checkpoint retreat and stale L1 snapshots.
- `op-supernode/supernode/activity/claimfollow/`: claim revocation, canonical
  recovery inputs, and the ambiguity between frontier retreat and invalidation.
- `op-acceptance-tests/tests/interop/private-interop/recovery_test.go`: real
  private execution and publication after an invalid interop message.
- `op-acceptance-tests/tests/interop/private-interop/outage_test.go`: recovery
  after sequencing-window expiry while the operator is stopped, followed by a
  fresh forced deposit whose message is published and relayed.
- `op-acceptance-tests/tests/interop/private-interop/l1_reorg_test.go`: canonical
  private L1 origins and resumed publication after a real L1 reorg.

The L1 reorg acceptance case removes a private origin. It does not isolate the
case where only a claim's L1 publication disappears while every private origin
stays canonical; the checkpoint-retreat unit test covers the adapter's behavior
at that boundary.

User deposits execute normally on both chains, including recovery positions. A
projection-only contract feature restricts protocol calls to the current batcher;
the execution client has no custom deposit no-op or receipt-log suppression rule.
See [the ETH profile](ETH-PROFILE.md) for the scope of those guards. Recovery obtains
the schedule from the supernode and deposit contents from its own L1 connection.
