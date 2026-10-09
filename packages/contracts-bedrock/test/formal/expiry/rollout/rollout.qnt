// -*- mode: Bluespec; -*-
// Rollout model of per-message interop expiry (exporter design). The messenger's expiry period is a
// storage value that initialize sets once (0 < period <= 365 days) and that upgrades read back, with a
// genesis override on test networks.
//
// Copied from ../quint/expiry.qnt and changed so that the single atomic `upgrade(x)` becomes separate,
// freely ordered activation events per chain and per layer:
//   - L1: the chain's L1CrossDomainMessenger implementation (old: no relayUndeliveredMessage; new:
//     relayUndeliveredMessage trusting the exporter, behind the INTEROP gate; earlier: the unreleased
//     37b44c48c7 design that trusted 0x..23), and the chain's SystemConfig INTEROP feature;
//   - L2: the L2ToL2CrossDomainMessenger implementation (old = 1.3.1, the one in the locked Lagoon
//     NUT bundle: no timestamps, no expireMessage, no L2CrossDomainMessenger target rule, HAS
//     resendMessage; new = 2.0.0 at the tip), the UndeliveredMessageExporter implementation
//     (none / the standard one / non-standard code), the SuperchainETHBridge (refundETH or not);
//   - the two NUT bundles as atomic L2 events: the tip bundle and the locked Lagoon bundle, the latter
//     subject to the L2ContractsManager's semver downgrade guard;
//   - lockbox joins at any point;
//   - the messenger's expiry period, set once: the production value when an upgrade first installs
//     the expiry messenger, an override at genesis on test networks (with the op-deployer and
//     devstack guards), and read back unchanged by every later upgrade;
//   - the protocol window W each relay is judged by, chosen adversarially per relay from a configured
//     set, separately before and after the destination's exporter first goes live (this covers W
//     changing over time, a W rule that is not yet active, and a host-supplied depset in kona's
//     fallback path).
//
// Each activation condition is a constant; the instances at the bottom drop them one at a time.
module rollout {
  // ---- Configuration ----
  // P_contract (days): expireMessage needs undeliveredAt > sentAt + P. It is the messenger's stored
  // expiry period, which initialize sets once: the production value
  // Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD (8) when an upgrade first installs the expiry messenger
  // or a genesis has no override. Later upgrades read the stored value back and keep it.
  const CONTRACT_PERIOD: int
  // The genesis override of the period (L2Genesis input l2ToL2MessageExpiryPeriod, set through
  // op-deployer's global override or the devstack preset):
  //   "none":      op-deployer refuses it for standard intents and for chains on public L1s;
  //   "guarded":   a test network launched with interop at genesis (L2Genesis and the devstack refuse
  //                it otherwise), with the devstack check that the period exceeds the message expiry
  //                window;
  //   "unguarded": the same without the window check.
  const PERIOD_OVERRIDE: str
  // Windows a relay on chain d may be judged by, before d's exporter has ever had an implementation,
  // and from then on. INF (no W rule) is a member when the rule is not enforced.
  const WIN_BEFORE: str -> Set[int]
  const WIN_AFTER: str -> Set[int]
  // L2 governance (the L2 ProxyAdmin owner, via ProxyAdmin.upgrade) may set the messenger back to
  // 1.3.1: "never", "beforeTimestamps" (only while no message from that chain has a timestamp), "any".
  const GOV_MSGR_DOWNGRADE: str
  // The L2ContractsManager refuses to lower a predeploy's semver (L2ContractsManagerUtils.upgradeTo).
  const L2CM_DOWNGRADE_GUARD: bool
  // L2 governance of a chain may set a non-standard implementation at the exporter address, while
  // the chain is a lockbox member / while it is outside the lockbox.
  const ROGUE_EXPORTER_MEMBER: bool
  const ROGUE_EXPORTER_OUTSIDER: bool
  // A chain joins the lockbox only if its exporter address never ran non-standard code.
  const JOIN_REQUIRES_CLEAN_EXPORTER: bool
  // An L1CrossDomainMessenger implementation of the earlier design (trusts 0x..23, no INTEROP gate)
  // may be installed on L1.
  const L1_EARLIER_DESIGN: bool
  // L2 chain IDs are unique among the chains that are or become lockbox members (else C has B's ID).
  const UNIQUE_IDS: bool
  // 2.0.0's sendMessage still writes sentMessages[nonce] = H (true at the tip), which is what 1.3.1's
  // resendMessage checks. False models the hardening "2.0.0 stops writing sentMessages".
  const NEW_WRITES_SENT_MESSAGES: bool

  // ---- World ----
  pure val MAX_TIME = 16
  pure val TIMES = 1.to(MAX_TIME)
  pure val INF = 99
  // The cap in op-core / kona / op-interop-filter: 7 days.
  pure val CAP = 7
  pure val CHAINS = Set("A", "B", "C")
  pure val INIT_CLUSTER = Set("A", "B")
  pure val MSGS = Set("m1", "m2", "m3")
  pure val SRC = Map("m1" -> "A", "m2" -> "B", "m3" -> "A")
  pure val DST = Map("m1" -> "B", "m2" -> "A", "m3" -> "C")
  pure val CHAIN_ID = if (UNIQUE_IDS) Map("A" -> "A", "B" -> "B", "C" -> "C")
                      else Map("A" -> "A", "B" -> "B", "C" -> "B")
  pure val L1_VERSIONS = if (L1_EARLIER_DESIGN) Set("old", "new", "earlier") else Set("old", "new")

  // Per-chain deployment state.
  type Chain = {
    l1: str,             // L1CrossDomainMessenger implementation: "old" | "new" | "earlier"
    l1Interop: bool,     // SystemConfig INTEROP (set by OPCM migrate; cannot be cleared)
    msgr: str,           // L2ToL2CrossDomainMessenger: "old" (1.3.1) | "new" (2.0.0)
    exporter: str,       // UndeliveredMessageExporter proxy implementation: "none" | "std" | "rogue"
    bridge: bool,        // SuperchainETHBridge has refundETH
    rogueEver: bool,     // history: the exporter address ever ran non-standard code
    exporterLive: bool,  // history: the exporter ever had an implementation (W switches to WIN_AFTER)
    hist: Set[str],      // ghost history, for witnesses
  }
  // inSentMessages: sentMessages[nonce] == H on the source (what 1.3.1 resendMessage requires).
  type Msg = { sentAt: int, events: Set[int], inSentMessages: bool, relayed: bool, expired: bool, refunded: bool, refunds: int }
  type Fact = { msg: str, dest: str, at: int, route: str, by: str, honest: bool }
  // An L2->L1 message from chain `from`, recorded sender `sender`; `l1Ready` records whether the
  // route's L1 accepted facts when the withdrawal was made (ghost).
  type Withdrawal = { from: str, sender: str, fact: Fact, l1Ready: bool }
  type Deposit = { to: str, fact: Fact }
  type Ghost = { forged: bool, edge: bool, lateBeforeExporter: bool, deferredL1: bool }

  // ---- State ----
  var now: str -> int
  // The stored expiry period per chain (0 = never initialized). Kept outside `Chain` so the record
  // type, which the flattened model repeats everywhere, stays small. 1.3.1 does not touch it, so it
  // survives a rollback and is read back by the next upgrade.
  var period: str -> int
  var chains: str -> Chain
  var cluster: Set[str]
  var msgs: str -> Msg
  var withdrawals: Set[Withdrawal]
  var deposits: Set[Deposit]
  var ghost: Ghost
  // (d, s): d's exporter got the standard implementation while s's messenger was 1.3.1.
  var early: Set[(str, str)]

  pure def within(evts: Set[int], t: int, w: int): bool = evts.exists(e => t <= e + w)
  def l1Accepts(x: str): bool = chains.get(x).l1 == "new" and chains.get(x).l1Interop

  action init = all {
    now' = CHAINS.mapBy(_ => 1),
    period' = CHAINS.mapBy(_ => 0),
    // Pre-rollout: every L2 runs the 1.3.1 messenger, the old bridge and no exporter implementation
    // (the genesis proxy reverts); every L1 runs the old messenger. A and B were migrated into the
    // shared lockbox (INTEROP set on L1); C is outside.
    chains' = CHAINS.mapBy(x => {
      l1: "old", l1Interop: INIT_CLUSTER.contains(x), msgr: "old", exporter: "none", bridge: false,
      rogueEver: false, exporterLive: false, hist: Set(),
    }),
    cluster' = INIT_CLUSTER,
    msgs' = MSGS.mapBy(_ => { sentAt: 0, events: Set(), inSentMessages: false, relayed: false, expired: false, refunded: false, refunds: 0 }),
    withdrawals' = Set(),
    deposits' = Set(),
    ghost' = { forged: false, edge: false, lateBeforeExporter: false, deferredL1: false },
    early' = Set(),
  }

  // ---- Frame helpers ----
  action keepMsgState = all { msgs' = msgs, withdrawals' = withdrawals, deposits' = deposits, ghost' = ghost }
  action setChain(x: str, c: Chain): bool = all {
    chains' = chains.set(x, c),
    now' = now, period' = period, cluster' = cluster, early' = early, keepMsgState,
  }
  // setChain that also installs x's expiry messenger: initialize sets the period p if it was never
  // set; otherwise the upgrade reads the stored period back and keeps it.
  action setChainP(x: str, c: Chain, p: int): bool = all {
    chains' = chains.set(x, c), period' = period.set(x, if (period.get(x) == 0) p else period.get(x)),
    now' = now, cluster' = cluster, early' = early, keepMsgState,
  }

  // ---------------------------------------------------------------------------------------------
  // Activation events. Each can happen on any chain, at any time, in any order.
  // ---------------------------------------------------------------------------------------------

  // L1: OPCM (or a rollback) sets the L1CrossDomainMessenger implementation.
  action l1Set(x: str, v: str): bool = {
    val c = chains.get(x)
    all {
      L1_VERSIONS.contains(v),
      c.l1 != v,
      setChain(x, c.with("l1", v).with("hist", if (c.l1 == "new") c.hist.union(Set("l1Down")) else c.hist)),
    }
  }

  // L1: SystemConfig INTEROP is enabled (it cannot be cleared).
  action l1EnableInterop(x: str): bool = {
    val c = chains.get(x)
    all { not(c.l1Interop), setChain(x, c.with("l1Interop", true)) }
  }

  // L2: the tip's NUT bundle (current-upgrade-bundle.json): the L2ContractsManager installs the
  // 2.0.0 messenger, the bridge with refundETH and the standard exporter in one transaction.
  action nutTip(x: str): bool = {
    val c = chains.get(x)
    all {
      setChainP(x, c.with("msgr", "new").with("bridge", true).with("exporter", "std")
                    .with("exporterLive", true).with("hist", c.hist.union(Set("tip"))), CONTRACT_PERIOD),
    }
  }

  // L2: the locked Lagoon NUT bundle (op-core/nuts/fork_lock.toml, commit fa9974a2): messenger 1.3.1
  // and the bridge without refundETH; it does not touch the exporter. With the downgrade guard, the
  // whole L2ContractsManager call reverts if either predeploy is already newer.
  action nutLagoon(x: str): bool = {
    val c = chains.get(x)
    val newer = c.msgr == "new" or c.bridge
    if (L2CM_DOWNGRADE_GUARD and newer)
      setChain(x, c.with("hist", c.hist.union(Set("lagoonReverted"))))
    else
      setChain(x, c.with("msgr", "old").with("bridge", false)
                   .with("hist", c.hist.union(if (newer) Set("lagoonDowngraded") else Set("lagoon"))))
  }

  // L2: one predeploy at a time (a split bundle, a later fork, or governance's ProxyAdmin.upgrade).
  action setMessengerNew(x: str): bool = {
    val c = chains.get(x)
    all { c.msgr == "old", setChainP(x, c.with("msgr", "new"), CONTRACT_PERIOD) }
  }
  // The finite windows any relay may be judged by once exporters are live: the configured window the
  // devstack compares an overridden period against.
  pure val MAX_WINDOW = CHAINS.fold(0, (acc, d) => WIN_AFTER.get(d).filter(w => w != INF)
                                                    .fold(acc, (a, w) => if (w > a) w else a))
  // A test network launched with interop at genesis, its messenger initialized with period p
  // (L2Genesis input l2ToL2MessageExpiryPeriod; 0 < p, and p <= 365 days, which exceeds MAX_TIME).
  // Genesis: x has no history and no message touches it. All of x's interop predeploys are live from
  // the start; later upgrades (nutTip, setMessengerNew) keep p.
  action genesisWithPeriod(x: str, p: int): bool = {
    val c = chains.get(x)
    all {
      PERIOD_OVERRIDE != "none",
      p > 0,
      PERIOD_OVERRIDE == "guarded" implies p > MAX_WINDOW,
      c.msgr == "old", c.hist == Set(), not(c.exporterLive), now.get(x) == 1,
      MSGS.forall(m => (SRC.get(m) == x or DST.get(m) == x) implies msgs.get(m).events == Set()),
      setChainP(x, c.with("msgr", "new").with("bridge", true).with("exporter", "std")
                    .with("exporterLive", true).with("hist", Set("genesisOverride")), p),
    }
  }
  // Governance rolls the messenger back to 1.3.1 (ProxyAdmin.upgrade has no version check).
  action govMessengerDowngrade(x: str): bool = {
    val c = chains.get(x)
    all {
      c.msgr == "new",
      GOV_MSGR_DOWNGRADE == "any" or
        (GOV_MSGR_DOWNGRADE == "beforeTimestamps" and MSGS.forall(m => SRC.get(m) == x implies msgs.get(m).sentAt == 0)),
      setChain(x, c.with("msgr", "old").with("hist", c.hist.union(Set("msgrDown")))),
    }
  }
  // The bridge in either direction (a rollback only removes refundETH; `refunded` stays in storage).
  action setBridge(x: str, b: bool): bool = {
    val c = chains.get(x)
    all {
      c.bridge != b,
      setChain(x, c.with("bridge", b).with("hist", if (b) c.hist else c.hist.union(Set("bridgeDown")))),
    }
  }
  // The exporter: the standard implementation, none (rolled back), or non-standard code.
  action setExporter(x: str, v: str): bool = {
    val c = chains.get(x)
    val member = cluster.contains(x)
    all {
      c.exporter != v,
      v == "rogue" implies (if (member) ROGUE_EXPORTER_MEMBER else ROGUE_EXPORTER_OUTSIDER),
      chains' = chains.set(x, c.with("exporter", v)
                               .with("rogueEver", c.rogueEver or v == "rogue")
                               .with("exporterLive", c.exporterLive or v != "none")
                               .with("hist", if (v == "none") c.hist.union(Set("exporterRemoved")) else c.hist)),
      early' = if (v == "std") early.union(CHAINS.filter(s => chains.get(s).msgr == "old").map(s => (x, s)))
               else early,
      now' = now, period' = period, cluster' = cluster, keepMsgState,
    }
  }

  // L1: x's portal is authorized in the shared ETHLockbox (ETHLockbox.authorizePortal or OPCM migrate).
  action join(x: str): bool = {
    val c = chains.get(x)
    all {
      not(cluster.contains(x)),
      JOIN_REQUIRES_CLEAN_EXPORTER implies not(c.rogueEver),
      cluster' = cluster.union(Set(x)),
      chains' = chains.set(x, c.with("hist", c.hist.union(Set("joined")))),
      now' = now, period' = period, early' = early, keepMsgState,
    }
  }

  // ---------------------------------------------------------------------------------------------
  // Protocol actions (as in ../quint/expiry.qnt, guarded by the per-chain deployment state).
  // ---------------------------------------------------------------------------------------------
  action keepDeploy = all { now' = now, period' = period, chains' = chains, cluster' = cluster, early' = early }
  action keepDeployAt(x: str, t: int): bool = all {
    now' = now.set(x, t), period' = period, chains' = chains, cluster' = cluster, early' = early,
  }

  // Clocks: the actions that read a chain's clock (send, resend, relay, exportStd) take the time t
  // they happen at, any t >= that chain's clock, and move the clock to t. This replaces a separate
  // tick action (clocks are read nowhere else), so traces need fewer steps.

  // sendMessage (SuperchainETHBridge.sendETH). Both messenger versions send; only 2.0.0 records sentAt.
  action send(m: str, t: int): bool = {
    val s = SRC.get(m)
    val ms = msgs.get(m)
    all {
      ms.events == Set(),
      now.get(s) <= t,
      msgs' = msgs.set(m, ms.with("events", Set(t))
                            .with("sentAt", if (chains.get(s).msgr == "new") t else 0)
                            .with("inSentMessages", chains.get(s).msgr == "old" or NEW_WRITES_SENT_MESSAGES)),
      withdrawals' = withdrawals, deposits' = deposits, ghost' = ghost, keepDeployAt(s, t),
    }
  }

  // resendMessage: exists only in 1.3.1. It re-emits SentMessage for any hash in sentMessages[nonce],
  // which both versions write, and never touches sentMessageTimestamps.
  action resend(m: str, t: int): bool = {
    val s = SRC.get(m)
    val ms = msgs.get(m)
    all {
      chains.get(s).msgr == "old",
      ms.events != Set(),
      ms.inSentMessages,
      now.get(s) <= t,
      msgs' = msgs.set(m, ms.with("events", ms.events.union(Set(t)))),
      withdrawals' = withdrawals, deposits' = deposits, ghost' = ghost, keepDeployAt(s, t),
    }
  }

  // relayMessage on the destination (both versions relay), valid iff some initiating event is
  // within the window w that the destination's cross-safety / fault proof applies to this relay.
  action relay(m: str, w: int, t: int): bool = {
    val d = DST.get(m)
    val ms = msgs.get(m)
    val live = chains.get(d).exporterLive
    all {
      not(ms.relayed),
      (if (live) WIN_AFTER.get(d) else WIN_BEFORE.get(d)).contains(w),
      now.get(d) <= t,
      within(ms.events, t, w),
      msgs' = msgs.set(m, ms.with("relayed", true)),
      ghost' = ghost.with("edge", ghost.edge or (live and ms.events.exists(e => t == e + CAP)))
                    .with("lateBeforeExporter", ghost.lateBeforeExporter or
                                                (not(live) and not(within(ms.events, t, CAP)))),
      withdrawals' = withdrawals, deposits' = deposits, keepDeployAt(d, t),
    }
  }

  // UndeliveredMessageExporter.exportUndeliveredMessage on chain x with the standard implementation.
  // It can name any message preimage (also one not sent yet) and any source messenger (route).
  action exportStd(x: str, m: str, route: str, t: int): bool = all {
    chains.get(x).exporter == "std",
    now.get(x) <= t,
    not(x == DST.get(m) and msgs.get(m).relayed),
    withdrawals' = withdrawals.union(Set({
      from: x, sender: "exporter", l1Ready: l1Accepts(route),
      fact: { msg: m, dest: CHAIN_ID.get(x), at: t, route: route, by: x, honest: true },
    })),
    msgs' = msgs, deposits' = deposits, ghost' = ghost, keepDeployAt(x, t),
  }

  // Non-standard code at the exporter address: withdrawals with the exporter as sender, any content.
  action exportRogue(x: str, m: str, d: str, t: int, route: str): bool = all {
    chains.get(x).exporter == "rogue",
    withdrawals' = withdrawals.union(Set({
      from: x, sender: "exporter", l1Ready: l1Accepts(route),
      fact: { msg: m, dest: d, at: t, route: route, by: x, honest: false },
    })),
    msgs' = msgs, deposits' = deposits, ghost' = ghost, keepDeploy,
  }

  // The 1.3.1 messenger has no L2CrossDomainMessenger target rule: an attacker relays a message to
  // L2CrossDomainMessenger.sendMessage(route's L1CDM, relayUndeliveredMessage(...)); recorded sender 0x..23.
  action messengerForge(x: str, m: str, d: str, t: int, route: str): bool = all {
    chains.get(x).msgr == "old",
    withdrawals' = withdrawals.union(Set({
      from: x, sender: "messenger", l1Ready: l1Accepts(route),
      fact: { msg: m, dest: d, at: t, route: route, by: x, honest: false },
    })),
    msgs' = msgs, deposits' = deposits, ghost' = ghost, keepDeploy,
  }

  // L1: the withdrawal is relayed into route's relayUndeliveredMessage. A failed L1 relay stays
  // replayable in the caller's failedMessages, so withdrawals are never consumed. The three checks
  // of the new implementation are as in ../quint/expiry.qnt (all on): a real messenger, in route's
  // lockbox now, recorded sender = the exporter; plus the INTEROP gate on route's SystemConfig.
  action l1Relay(w: Withdrawal): bool = {
    val r = w.fact.route
    val c = chains.get(r)
    all {
      withdrawals.contains(w),
      w.from != r,
      cluster.contains(w.from),
      (c.l1 == "new" and c.l1Interop and w.sender == "exporter") or (c.l1 == "earlier" and w.sender == "messenger"),
      deposits' = deposits.union(Set({ to: r, fact: w.fact })),
      ghost' = ghost.with("deferredL1", ghost.deferredL1 or not(w.l1Ready)),
      msgs' = msgs, withdrawals' = withdrawals, keepDeploy,
    }
  }

  // expireMessage on `to` (2.0.0 only; on 1.3.1 the call reverts and the L2CrossDomainMessenger keeps
  // it replayable, so deposits are never consumed).
  action expire(dep: Deposit): bool = {
    val f = dep.fact
    val ms = msgs.get(f.msg)
    all {
      deposits.contains(dep),
      chains.get(dep.to).msgr == "new",
      dep.to == SRC.get(f.msg),
      f.dest == CHAIN_ID.get(DST.get(f.msg)),
      ms.sentAt != 0,
      f.at > ms.sentAt + period.get(dep.to),
      msgs' = msgs.set(f.msg, ms.with("expired", true)),
      ghost' = ghost.with("forged", ghost.forged or not(f.honest and f.by == DST.get(f.msg))),
      withdrawals' = withdrawals, deposits' = deposits, keepDeploy,
    }
  }

  // refundETH on the source: needs the new bridge and a messenger with expiredMessages (2.0.0).
  action refund(m: str): bool = {
    val s = SRC.get(m)
    val ms = msgs.get(m)
    all {
      chains.get(s).bridge, chains.get(s).msgr == "new",
      ms.expired, not(ms.refunded),
      msgs' = msgs.set(m, ms.with("refunded", true).with("refunds", ms.refunds + 1)),
      withdrawals' = withdrawals, deposits' = deposits, ghost' = ghost, keepDeploy,
    }
  }

  action step = {
    nondet m = MSGS.oneOf()
    nondet x = CHAINS.oneOf()
    nondet d = CHAINS.oneOf()
    nondet route = CHAINS.oneOf()
    nondet t = TIMES.oneOf()
    nondet w = WIN_BEFORE.get(x).union(WIN_AFTER.get(x)).oneOf()
    nondet v1 = Set("old", "new", "earlier").oneOf()
    nondet ve = Set("none", "std", "rogue").oneOf()
    nondet b = Set(true, false).oneOf()
    nondet p = TIMES.oneOf()
    any {
      l1Set(x, v1),
      l1EnableInterop(x),
      nutTip(x),
      nutLagoon(x),
      setMessengerNew(x),
      genesisWithPeriod(x, p),
      govMessengerDowngrade(x),
      setBridge(x, b),
      setExporter(x, ve),
      join(x),
      send(m, t),
      resend(m, t),
      // The window set is per destination; `x` only picks which chain's set w is drawn from.
      all { x == DST.get(m), relay(m, w, t) },
      exportStd(x, m, route, t),
      exportRogue(x, m, d, t, route),
      messengerForge(x, m, d, t, route),
      all { withdrawals != Set(), nondet wd = withdrawals.oneOf() l1Relay(wd) },
      all { deposits != Set(), nondet dep = deposits.oneOf() expire(dep) },
      refund(m),
    }
  }

  // ---- Properties ----
  val NoDoubleSpend = MSGS.forall(m => not(msgs.get(m).relayed and msgs.get(m).refunded))
  val RefundImpliesExpired = MSGS.forall(m => msgs.get(m).refunded implies msgs.get(m).expired)
  // An expired message was never relayed, and no initiating event is within any post-exporter
  // window of the destination's clock.
  val ExpiredImpliesNeverRelayable = MSGS.forall(m => {
    val d = DST.get(m)
    msgs.get(m).expired implies (not(msgs.get(m).relayed) and
      WIN_AFTER.get(d).forall(w => not(within(msgs.get(m).events, now.get(d), w))))
  })
  val NoForgedFact = not(ghost.forged)
  val AtMostOneRefund = MSGS.forall(m => msgs.get(m).refunds <= 1)
  // `Safety` is what the deep checks use. ExpiredImpliesNeverRelayable (a stronger auxiliary
  // invariant, and by far the most expensive for the solver) is in `SafetyFull`, which is checked to a
  // smaller depth.
  val Safety = NoDoubleSpend and RefundImpliesExpired and NoForgedFact and AtMostOneRefund
  val SafetyFull = Safety and ExpiredImpliesNeverRelayable

  // ---- Non-vacuity witnesses (each must be VIOLATED in the safe instances) ----
  def refundedWith(p: str => bool): bool = MSGS.exists(m => msgs.get(m).refunded and p(m))
  val NoRefundEver = not(refundedWith(_ => true))
  val NoRefundOfM2 = not(msgs.get("m2").refunded)
  // m3's destination C joined the lockbox mid-rollout.
  val NoRefundOfM3 = not(msgs.get("m3").refunded)
  // A relay at exactly exec - init == 7 after the destination's exporter went live.
  val NoEdgeRelay = not(ghost.edge)
  // A relay with exec - init > 7 before the destination's exporter went live (no W rule yet).
  val NoLateRelayBeforeExporter = not(ghost.lateBeforeExporter)
  // A fact made while its route's L1 did not accept facts is accepted by that L1 later.
  val NoDeferredL1Fact = not(ghost.deferredL1)
  // A refund of m whose destination's exporter went live while m's source still ran 1.3.1.
  val NoRefundExporterFirst = not(refundedWith(m => early.contains((DST.get(m), SRC.get(m)))))
  // Refunds on a source that also saw: the Lagoon bundle applied (1.3.1) / the Lagoon bundle after
  // the tip (reverted by the guard) / an L1 rollback / a bridge rollback; and on a destination whose
  // exporter was rolled back to none.
  val NoRefundAfterLagoon = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("lagoon")))
  val NoRefundAfterLagoonReverted = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("lagoonReverted")))
  val NoRefundAfterL1Rollback = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("l1Down")))
  val NoRefundAfterBridgeRollback = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("bridgeDown")))
  val NoRefundAfterExporterRemoved = not(refundedWith(m => chains.get(DST.get(m)).hist.contains("exporterRemoved")))
  // A refund on a source whose messenger was rolled back to 1.3.1 (before any timestamp) and re-upgraded.
  val NoRefundAfterMessengerRollback = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("msgrDown")))
  // A refund on a source launched at genesis with an overridden expiry period.
  val NoRefundUnderPeriodOverride = not(refundedWith(m => chains.get(SRC.get(m)).hist.contains("genesisOverride")))
  // Each remaining activation event and attacker move occurs (reachability only):
  // a lockbox join; INTEROP enabled on L1 for a chain that started without it; non-standard exporter
  // code on some chain (only outsiders in rolloutSafe); a withdrawal from such code; a 0x..23
  // withdrawal through a 1.3.1 messenger without the target rule; a resend (1.3.1).
  val NoJoin = cluster == INIT_CLUSTER
  val NoLateInteropEnable = not(chains.get("C").l1Interop)
  val NoRogueExporterEver = not(CHAINS.exists(x => chains.get(x).rogueEver))
  val NoRogueWithdrawal = not(withdrawals.exists(w => w.sender == "exporter" and not(w.fact.honest)))
  val NoMessengerWithdrawal = not(withdrawals.exists(w => w.sender == "messenger"))
  val NoResend = not(MSGS.exists(m => msgs.get(m).events.exists(e1 => msgs.get(m).events.exists(e2 => e1 != e2))))

  // ---- Literals for the scripted traces in the instances ----
  pure def fact(m: str, dest: str, at: int, route: str, by: str, honest: bool): Fact =
    { msg: m, dest: dest, at: at, route: route, by: by, honest: honest }
  pure def wd(src: str, sender: str, f: Fact, ready: bool): Withdrawal =
    { from: src, sender: sender, fact: f, l1Ready: ready }
  pure def dep(dst: str, f: Fact): Deposit = { to: dst, fact: f }
  // The honest path for m1 (A -> B) once A's messenger and bridge and B's exporter are live and A's
  // L1 accepts facts: send at A-time 1, export at B-time 10, relay to L1, expire, refund.
  pure val F1 = fact("m1", "B", 10, "A", "B", true)
  run honestTail1 = send("m1", 1).then(exportStd("B", "m1", "A", 10))
    .then(l1Relay(wd("B", "exporter", F1, true))).then(expire(dep("A", F1))).then(refund("m1"))
}

// ---------------------------------------------------------------------------------------------
// Instances. Days. "rollout" is the design with every activation condition kept and every
// ordering enabled; the others drop one condition each.
// ---------------------------------------------------------------------------------------------

// All conditions. Before a destination's exporter is ever live its relays may be judged by any
// window, including none (INF); from then on by 5 or 7 days per relay (the cap; covers W changing and
// a host-chosen depset). Outsiders may run anything at the exporter address.
module rolloutSafe {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").*

  // Reachability witnesses as scripted traces (`quint test --main=rolloutSafe`); each ends with
  // Safety holding and the named witness violated.
  pure val F2 = fact("m2", "A", 10, "B", "A", true)
  pure val F3 = fact("m3", "C", 10, "A", "C", true)
  run witnessHonestRefund = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(honestTail1).expect(SafetyFull and not(NoRefundEver))
  run witnessRefundM2 = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("B", "new"))
    .then(send("m2", 1)).then(exportStd("A", "m2", "B", 10))
    .then(l1Relay(wd("A", "exporter", F2, true))).then(expire(dep("B", F2))).then(refund("m2"))
    .expect(SafetyFull and not(NoRefundOfM2))
  run witnessRefundM3JoinMidRollout = init.then(nutTip("A")).then(nutTip("C")).then(l1Set("A", "new"))
    .then(send("m3", 1)).then(exportStd("C", "m3", "A", 10)).then(join("C"))
    .then(l1Relay(wd("C", "exporter", F3, true))).then(expire(dep("A", F3))).then(refund("m3"))
    .expect(SafetyFull and not(NoRefundOfM3))
  run witnessEdgeRelay = init.then(nutTip("A")).then(nutTip("B")).then(send("m1", 1))
    .then(relay("m1", 7, 8)).expect(SafetyFull and not(NoEdgeRelay))
  run witnessLateRelayBeforeExporter = init.then(send("m1", 1)).then(relay("m1", 99, 10))
    .then(nutTip("B")).expect(SafetyFull and not(NoLateRelayBeforeExporter))
  run witnessDeferredL1 = init.then(nutTip("A")).then(nutTip("B")).then(send("m1", 1))
    .then(exportStd("B", "m1", "A", 10)).then(l1Set("A", "new"))
    .then(l1Relay(wd("B", "exporter", F1, false))).then(expire(dep("A", F1))).then(refund("m1"))
    .expect(SafetyFull and not(NoDeferredL1Fact))
  run witnessExporterFirst = init.then(setExporter("B", "std")).then(setMessengerNew("A"))
    .then(setBridge("A", true)).then(l1Set("A", "new")).then(honestTail1)
    .expect(SafetyFull and not(NoRefundExporterFirst))
  run witnessLagoonThenTip = init.then(nutLagoon("A")).then(nutTip("A")).then(nutTip("B"))
    .then(l1Set("A", "new")).then(honestTail1).expect(SafetyFull and not(NoRefundAfterLagoon))
  run witnessLagoonAfterTipReverts = init.then(nutTip("A")).then(nutLagoon("A"))
    .expect(chains.get("A").msgr == "new" and chains.get("A").bridge)
    .then(nutTip("B")).then(l1Set("A", "new")).then(honestTail1)
    .expect(SafetyFull and not(NoRefundAfterLagoonReverted))
  run witnessL1Rollback = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(l1Set("A", "old")).then(l1Set("A", "new")).then(honestTail1)
    .expect(SafetyFull and not(NoRefundAfterL1Rollback))
  run witnessBridgeRollback = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(setBridge("A", false)).then(setBridge("A", true)).then(honestTail1)
    .expect(SafetyFull and not(NoRefundAfterBridgeRollback))
  run witnessExporterRemoved = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(send("m1", 1)).then(exportStd("B", "m1", "A", 10)).then(setExporter("B", "none"))
    .then(l1Relay(wd("B", "exporter", F1, true))).then(expire(dep("A", F1))).then(refund("m1"))
    .expect(SafetyFull and not(NoRefundAfterExporterRemoved))
  run witnessMessengerRollbackBeforeTimestamps = init.then(nutTip("A")).then(govMessengerDowngrade("A"))
    .then(setMessengerNew("A")).then(nutTip("B")).then(l1Set("A", "new")).then(honestTail1)
    .expect(SafetyFull and not(NoRefundAfterMessengerRollback))
  // What the conditions rule out: no governance downgrade after a timestamp; the Lagoon bundle after the tip leaves
  // 2.0.0 in place, so resend stays unavailable; no window above 7 after the exporter; no join of a
  // chain whose exporter ran non-standard code; the earlier-design L1 is not installable.
  // The remaining witnesses as traces: INTEROP enabled later on L1 (C); a 0x..23 withdrawal from a
  // 1.3.1 messenger, which L1 rejects; non-standard exporter code on an outsider and its withdrawal,
  // which L1 rejects (C is not a member); a resend on 1.3.1 (no timestamp, so it never expires).
  run witnessLateInteropEnable = init.then(l1EnableInterop("C")).expect(SafetyFull and not(NoLateInteropEnable))
  run witnessMessengerWithdrawalRejected = init.then(nutTip("A")).then(l1Set("A", "new"))
    .then(messengerForge("B", "m1", "B", 10, "A"))
    .expect(SafetyFull and not(NoMessengerWithdrawal))
    .then(l1Relay(wd("B", "messenger", fact("m1", "B", 10, "A", "B", false), true)).fail())
  run witnessRogueOutsiderRejected = init.then(nutTip("A")).then(l1Set("A", "new")).then(setExporter("C", "rogue"))
    .then(exportRogue("C", "m1", "B", 10, "A"))
    .expect(SafetyFull and not(NoRogueExporterEver) and not(NoRogueWithdrawal))
    .then(l1Relay(wd("C", "exporter", fact("m1", "B", 10, "A", "C", false), true)).fail())
  run witnessResendOn131 = init.then(send("m1", 1)).then(resend("m1", 3))
    .expect(SafetyFull and not(NoResend) and msgs.get("m1").sentAt == 0)
  run blockedGovDowngrade = init.then(nutTip("A")).then(send("m1", 1)).then(govMessengerDowngrade("A").fail())
  run blockedResendAfterLagoon = init.then(nutTip("A")).then(send("m1", 1)).then(nutLagoon("A"))
    .then(resend("m1", 3).fail())
  run blockedWideWindow = init.then(nutTip("A")).then(nutTip("B")).then(send("m1", 1))
    .then(relay("m1", 99, 10).fail())
  run blockedRogueJoin = init.then(setExporter("C", "rogue")).then(join("C").fail())
  run blockedEarlierL1 = init.then(l1Set("A", "earlier").fail())
  // A source launched with an overridden period above the window: a fact at 10 = 1 + 9 does not
  // expire the message; one at 11 does, and the refund goes through.
  pure val F1At10 = fact("m1", "B", 10, "A", "B", true)
  pure val F1At11 = fact("m1", "B", 11, "A", "B", true)
  run witnessPeriodOverride = init.then(genesisWithPeriod("A", 9)).then(nutTip("B")).then(l1Set("A", "new"))
    .then(send("m1", 1)).then(exportStd("B", "m1", "A", 11))
    .then(l1Relay(wd("B", "exporter", F1At11, true))).then(expire(dep("A", F1At11))).then(refund("m1"))
    .expect(SafetyFull and not(NoRefundUnderPeriodOverride))
  run blockedExpireWithinOverriddenPeriod = init.then(genesisWithPeriod("A", 9)).then(nutTip("B"))
    .then(l1Set("A", "new")).then(send("m1", 1)).then(exportStd("B", "m1", "A", 10))
    .then(l1Relay(wd("B", "exporter", F1At10, true))).then(expire(dep("A", F1At10)).fail())
  // The devstack guard: a period not above the window (7) is refused; an override needs genesis.
  run blockedPeriodAtWindow = init.then(genesisWithPeriod("A", 7).fail())
  run blockedPeriodAfterGenesis = init.then(send("m1", 1)).then(genesisWithPeriod("A", 9).fail())
  // The period survives later upgrades: the tip bundle reads it back.
  run witnessPeriodReadBack = init.then(genesisWithPeriod("A", 9)).then(nutTip("A"))
    .expect(period.get("A") == 9)
  // The first install of the expiry messenger initializes the production period, and a rollback to
  // 1.3.1 and a new upgrade keep it.
  run witnessPeriodInitializedOnce = init.then(nutTip("A")).expect(period.get("A") == 8)
    .then(govMessengerDowngrade("A")).then(setMessengerNew("A")).expect(period.get("A") == 8)
}

// The bound is W <= P, not W <= 7: W = P = 8 after the exporter is still safe (strict `>`).
module rolloutWindowAtP {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(8), "B" -> Set(8), "C" -> Set(8)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").*

  run witnessHonestRefund = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(honestTail1).expect(SafetyFull and not(NoRefundEver))
  // A relay at exactly W = P = 8 after send, with B's exporter live: still before any fact could
  // expire the message (that needs undeliveredAt > sentAt + 8).
  run witnessRelayAtP = init.then(nutTip("A")).then(nutTip("B")).then(send("m1", 1))
    .then(relay("m1", 8, 9)).expect(SafetyFull and msgs.get("m1").relayed).then(exportStd("B", "m1", "A", 9).fail())
}

// Hardening check: if 2.0.0 stopped writing sentMessages, any rollback of the messenger to 1.3.1
// (governance, or the Lagoon bundle without the guard) would be safe: 1.3.1 can then only resend
// messages it sent itself, which have no timestamp and never expire.
module downgradeHardened {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "any", L2CM_DOWNGRADE_GUARD = false, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = false,
    PERIOD_OVERRIDE = "guarded").*
  // The govMessengerDowngrade counterexample's prefix: the resend is now impossible.
  run witnessDowngradeCannotResend = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(honestTail1).then(govMessengerDowngrade("A"))
    .expect(SafetyFull and not(NoRefundEver)).then(resend("m1", 3).fail())
  // 1.3.1 still resends what it sent itself (sentAt 0, never expires).
  run witnessOldSendResend = init.then(nutTip("A")).then(govMessengerDowngrade("A")).then(send("m1", 1))
    .then(resend("m1", 3)).expect(SafetyFull and msgs.get("m1").events == Set(1, 3))
}
