// -*- mode: Bluespec; -*-
// Protocol model of per-message interop expiry:
//   - L2ToL2CrossDomainMessenger records each message's send timestamp and marks messages expired;
//   - UndeliveredMessageExporter (a predeploy whose proxy has no implementation before the upgrade,
//     so nothing can be sent from its address) exports "message H was not relayed here by t" through the L2CrossDomainMessenger;
//   - the source chain's L1CrossDomainMessenger.relayUndeliveredMessage checks the withdrawal and
//     deposits expireMessage(H, t) to the source chain;
//   - SuperchainETHBridge.refundETH pays back an expired send.
//
// The model is at the level of "who can make which withdrawal / deposit exist": every L2->L1
// message carries the L2 address that sent it (the L2CrossDomainMessenger records msg.sender), and
// the attacker's moves are expressed per target contract, so that "only the exporter can make the
// trusted sender speak" is a consequence of the model, not an assumption.
//
// Each mitigation is a constant; the instances at the bottom switch them off one at a time.
module expiry {
  // ---- Design and mitigations ----
  // L2 sender that L1CrossDomainMessenger.relayUndeliveredMessage trusts: "exporter" (the design)
  // or "messenger" (the earlier design, which trusted the L2ToL2CrossDomainMessenger itself).
  const TRUSTED_SENDER: str
  // After its upgrade, the L2ToL2CrossDomainMessenger refuses to relay (and send) messages whose
  // target is the L2CrossDomainMessenger. Before the upgrade it has no such rule.
  const TARGET_RULE: bool
  // The three checks in relayUndeliveredMessage.
  const REAL_MESSENGER_CHECK: bool   // caller.portal().systemConfig().l1CrossDomainMessenger() == caller
  const LOCKBOX_CHECK: bool          // this chain's ETHLockbox authorizes caller.portal() (read at relay time)
  const SENDER_CHECK: bool           // caller.xDomainMessageSender() == TRUSTED_SENDER
  // Governance assumption: a chain is authorized in a lockbox only if its whole history ran the
  // standard predeploy code (so the exporter address never had other code on it).
  const JOIN_REQUIRES_STANDARD: bool
  // W_protocol: the relay validity rule (exec - init <= W), capped at 7 days.
  // P_contract: expireMessage's period (8 days). Safety needs P_contract >= W_protocol.
  const PROTOCOL_WINDOW: int
  const CONTRACT_PERIOD: int
  // expireMessage uses `>=` instead of `>` (boundary mutation).
  const EXPIRE_GE: bool
  // The removed resendMessage, and whether it would restart the stored timestamp.
  const RESEND: bool
  const RESEND_RESTARTS: bool
  // Chain IDs are unique among the chains that are or become lockbox members. If false, chain C
  // has the same L2 chain ID as B (a governance failure: two authorized histories, one ID).
  const UNIQUE_IDS: bool
  // An attacker can relay to the L2CrossDomainMessenger on a chain before its upgrade (true in
  // reality; switched off only to isolate the post-upgrade path in one counterexample).
  const PRE_UPGRADE_FORGERY: bool

  // ---- World ----
  // Time unit: one day. Clocks are per chain and only move forward.
  pure val MAX_TIME = 20
  pure val TIMES = 1.to(MAX_TIME)
  // A, B, C run the standard code. D runs arbitrary code. A and B start in the lockbox.
  pure val CHAINS = Set("A", "B", "C", "D")
  pure val STANDARD = Set("A", "B", "C")
  pure val INIT_CLUSTER = Set("A", "B")
  // Bridge sends with different sources (so facts must be routed to the right source), and one to
  // C, which is outside the lockbox at genesis and may join later.
  pure val MSGS = Set("m1", "m2", "m3")
  pure val SRC = Map("m1" -> "A", "m2" -> "B", "m3" -> "A")
  pure val DST = Map("m1" -> "B", "m2" -> "A", "m3" -> "C")
  // block.chainid of each chain.
  pure val CHAIN_ID = if (UNIQUE_IDS) Map("A" -> "A", "B" -> "B", "C" -> "C", "D" -> "D")
                      else Map("A" -> "A", "B" -> "B", "C" -> "B", "D" -> "D")
  pure val SENDERS = Set("messenger", "exporter", "user")

  // relayUndeliveredMessage(H, at) where H was computed with chain ID `dest` as the destination; `route` is
  // the chain whose L1CrossDomainMessenger the withdrawal targets; `honest` is true iff the exporter
  // (or, in the earlier design, the messenger's export function) produced it on chain `by`.
  type Fact = { msg: str, dest: str, at: int, route: str, by: str, honest: bool }
  // An L2->L1 message from chain `from`, sent through the L2CrossDomainMessenger by `sender`.
  type Withdrawal = { from: str, sender: str, fact: Fact }
  // An expireMessage(H, at) deposit sent by chain `to`'s L1CrossDomainMessenger to chain `to`.
  type Deposit = { to: str, fact: Fact }

  // ---- State ----
  var now: str -> int
  var upgraded: str -> bool
  var cluster: Set[str]
  // L2ToL2CrossDomainMessenger.sentMessageTimestamps on the message's source (0 = none).
  var sentAt: str -> int
  // Source-chain timestamps of the message's initiating SentMessage events.
  var events: str -> Set[int]
  // successfulMessages on the message's destination.
  var relayed: str -> bool
  // Withdrawals never expire on L1, and a failed L1 relay can be replayed, so they stay.
  var withdrawals: Set[Withdrawal]
  // Failed L2 deposits can be replayed, so they stay.
  var deposits: Set[Deposit]
  var expired: str -> bool
  var refunded: str -> bool
  var refunds: str -> int
  // Ghosts.
  var forgedAccepted: bool
  var edgeRelay: bool
  // Defense-in-depth ghost: an upgraded messenger (0x..23) initiated a withdrawal.
  var messengerSpokeAfterUpgrade: bool

  pure def withinWindow(evts: Set[int], t: int): bool = evts.exists(e => t <= e + PROTOCOL_WINDOW)
  pure def expiredBy(sent: int, at: int): bool =
    if (EXPIRE_GE) at >= sent + CONTRACT_PERIOD else at > sent + CONTRACT_PERIOD

  action init = all {
    now' = CHAINS.mapBy(_ => 1),
    upgraded' = CHAINS.mapBy(_ => false),
    cluster' = INIT_CLUSTER,
    sentAt' = MSGS.mapBy(_ => 0),
    events' = MSGS.mapBy(_ => Set()),
    relayed' = MSGS.mapBy(_ => false),
    withdrawals' = Set(),
    deposits' = Set(),
    expired' = MSGS.mapBy(_ => false),
    refunded' = MSGS.mapBy(_ => false),
    refunds' = MSGS.mapBy(_ => 0),
    forgedAccepted' = false,
    edgeRelay' = false,
    messengerSpokeAfterUpgrade' = false,
  }

  // A chain's clock moves forward (by any amount).
  action tick(x: str, t: int): bool = all {
    now.get(x) < t,
    now' = now.set(x, t),
    upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events, relayed' = relayed,
    withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // The network upgrade on a standard chain: the exporter's proxy (present since genesis, reverting
  // without an implementation) gets its implementation; the messenger gets the timestamp record and
  // (with TARGET_RULE) the target rule. Assumption: no ProxyAdmin action set an implementation at the
  // exporter address before the upgrade.
  action upgrade(x: str): bool = all {
    STANDARD.contains(x),
    not(upgraded.get(x)),
    upgraded' = upgraded.set(x, true),
    now' = now, cluster' = cluster, sentAt' = sentAt, events' = events, relayed' = relayed,
    withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // A chain's portal is authorized in the lockbox.
  action join(x: str): bool = all {
    not(cluster.contains(x)),
    JOIN_REQUIRES_STANDARD implies STANDARD.contains(x),
    cluster' = cluster.union(Set(x)),
    now' = now, upgraded' = upgraded, sentAt' = sentAt, events' = events, relayed' = relayed,
    withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // Source: sendMessage (SuperchainETHBridge.sendETH). Nonces are unique, so once per message.
  // Only the upgraded messenger records the timestamp.
  action send(m: str): bool = {
    val s = SRC.get(m)
    all {
      events.get(m) == Set(),
      sentAt' = if (upgraded.get(s)) sentAt.set(m, now.get(s)) else sentAt,
      events' = events.set(m, Set(now.get(s))),
      now' = now, upgraded' = upgraded, cluster' = cluster, relayed' = relayed,
      withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
      refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
    }
  }

  // resendMessage. The pre-upgrade messenger still has it (it re-emits the event; it records no
  // timestamp, since the old code has none). After the upgrade it exists only in the historical
  // variants (RESEND).
  action resend(m: str): bool = {
    val s = SRC.get(m)
    all {
      RESEND or not(upgraded.get(s)),
      events.get(m) != Set(),
      RESEND_RESTARTS implies not(expired.get(m)),
      events' = events.set(m, events.get(m).union(Set(now.get(s)))),
      sentAt' = if (RESEND_RESTARTS and upgraded.get(s)) sentAt.set(m, now.get(s)) else sentAt,
      now' = now, upgraded' = upgraded, cluster' = cluster, relayed' = relayed,
      withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
      refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
    }
  }

  // Destination: relayMessage, valid iff some initiating event is within the protocol window
  // (cross-safety and the fault proof reject exec - init > W).
  action relay(m: str): bool = {
    val d = DST.get(m)
    all {
      not(relayed.get(m)),
      withinWindow(events.get(m), now.get(d)),
      relayed' = relayed.set(m, true),
      edgeRelay' = (edgeRelay or events.get(m).exists(e => now.get(d) == e + PROTOCOL_WINDOW)),
      messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
      now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
      withdrawals' = withdrawals, deposits' = deposits, expired' = expired, refunded' = refunded,
      refunds' = refunds, forgedAccepted' = forgedAccepted,
    }
  }

  // Chain x, after its upgrade: the export function (the exporter predeploy, or the messenger in
  // the earlier design) with message m's preimage and any source messenger `route`. The hash uses x
  // as the destination; the export reverts if x relayed that hash.
  action exportUndelivered(x: str, m: str, route: str): bool = all {
    STANDARD.contains(x),
    upgraded.get(x),
    not(x == DST.get(m) and relayed.get(m)),
    withdrawals' = withdrawals.union(Set({
      from: x,
      sender: TRUSTED_SENDER,
      fact: { msg: m, dest: CHAIN_ID.get(x), at: now.get(x), route: route, by: x, honest: true },
    })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // Attacker on a standard chain x: relay a message of its own whose target is the
  // L2CrossDomainMessenger with calldata sendMessage(route's L1CDM, relayUndeliveredMessage(...)).
  // The L2CrossDomainMessenger records the L2ToL2CrossDomainMessenger as the sender. Possible before
  // x's upgrade (no target rule yet), and after it only without TARGET_RULE.
  // (Any other target cannot make 0x..23 the caller of the L2CrossDomainMessenger; the exporter
  // never calls a caller-chosen target, so no relay can make it speak.)
  action relayToL2CrossDomainMessenger(x: str, m: str, d: str, t: int, route: str): bool = all {
    STANDARD.contains(x),
    (PRE_UPGRADE_FORGERY and not(upgraded.get(x))) or (upgraded.get(x) and not(TARGET_RULE)),
    withdrawals' = withdrawals.union(Set({
      from: x,
      sender: "messenger",
      fact: { msg: m, dest: d, at: t, route: route, by: x, honest: false },
    })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = (messengerSpokeAfterUpgrade or upgraded.get(x)),
  }

  // Anyone on any chain: L2CrossDomainMessenger.sendMessage with any payload; the recorded sender
  // is the caller (a user or contract), never a predeploy.
  action userWithdrawal(x: str, m: str, d: str, t: int, route: str): bool = all {
    withdrawals' = withdrawals.union(Set({
      from: x,
      sender: "user",
      fact: { msg: m, dest: d, at: t, route: route, by: x, honest: false },
    })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // A chain running arbitrary code can produce withdrawals with any recorded sender. Only while it is
  // outside the lockbox, so that a counterexample through it shows withdrawals made before a join.
  action arbitraryCode(x: str, sender: str, m: str, d: str, t: int, route: str): bool = all {
    not(STANDARD.contains(x)),
    not(cluster.contains(x)),
    withdrawals' = withdrawals.union(Set({
      from: x,
      sender: sender,
      fact: { msg: m, dest: d, at: t, route: route, by: x, honest: false },
    })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, deposits' = deposits, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // L1: the withdrawal is proven, finalized and relayed by `from`'s L1CrossDomainMessenger into
  // `route`'s L1CrossDomainMessenger.relayUndeliveredMessage, which deposits expireMessage to route.
  action l1Relay(w: Withdrawal): bool = all {
    withdrawals.contains(w),
    // L1CDM._isUnsafeTarget: a messenger never relays to itself.
    w.from != w.fact.route,
    LOCKBOX_CHECK implies cluster.contains(w.from),
    SENDER_CHECK implies w.sender == TRUSTED_SENDER,
    deposits' = deposits.union(Set({ to: w.fact.route, fact: w.fact })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, withdrawals' = withdrawals, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // L1: any contract F calls route's relayUndeliveredMessage directly, pretending to be a messenger.
  // F answers every getter itself. If F names a real, authorized portal as its own, check (a) fails:
  // that portal's SystemConfig names the real messenger, not F. If F names a fake portal whose fake
  // SystemConfig names F, check (a) passes, but the fake portal is not in the lockbox, so check (b)
  // fails. Check (c) always passes (F returns the trusted sender). So F succeeds iff (a) or (b) is off.
  action fakeCaller(route: str, m: str, d: str, t: int): bool = all {
    not(REAL_MESSENGER_CHECK) or not(LOCKBOX_CHECK),
    deposits' = deposits.union(Set({
      to: route,
      fact: { msg: m, dest: d, at: t, route: route, by: "L1", honest: false },
    })),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, withdrawals' = withdrawals, expired' = expired, refunded' = refunded,
    refunds' = refunds, forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  // Chain `to`: the deposit runs expireMessage. Only the message's source has a timestamp for its
  // hash, and the hash also binds the destination.
  action expire(dep: Deposit): bool = {
    val f = dep.fact
    all {
      deposits.contains(dep),
      dep.to == SRC.get(f.msg),
      f.dest == CHAIN_ID.get(DST.get(f.msg)),
      sentAt.get(f.msg) != 0,
      expiredBy(sentAt.get(f.msg), f.at),
      expired' = expired.set(f.msg, true),
      forgedAccepted' = (forgedAccepted or not(f.honest and f.by == DST.get(f.msg))),
      now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
      relayed' = relayed, withdrawals' = withdrawals, deposits' = deposits, refunded' = refunded,
      refunds' = refunds, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
    }
  }

  // Source: SuperchainETHBridge.refundETH.
  action refund(m: str): bool = all {
    expired.get(m),
    not(refunded.get(m)),
    refunded' = refunded.set(m, true),
    refunds' = refunds.set(m, refunds.get(m) + 1),
    now' = now, upgraded' = upgraded, cluster' = cluster, sentAt' = sentAt, events' = events,
    relayed' = relayed, withdrawals' = withdrawals, deposits' = deposits, expired' = expired,
    forgedAccepted' = forgedAccepted, edgeRelay' = edgeRelay, messengerSpokeAfterUpgrade' = messengerSpokeAfterUpgrade,
  }

  action step = {
    nondet m = MSGS.oneOf()
    nondet x = CHAINS.oneOf()
    nondet d = CHAINS.oneOf()
    nondet route = CHAINS.oneOf()
    nondet t = TIMES.oneOf()
    nondet sender = SENDERS.oneOf()
    any {
      tick(x, t),
      upgrade(x),
      join(x),
      send(m),
      resend(m),
      relay(m),
      exportUndelivered(x, m, route),
      relayToL2CrossDomainMessenger(x, m, d, t, route),
      userWithdrawal(x, m, d, t, route),
      arbitraryCode(x, sender, m, d, t, route),
      all { withdrawals != Set(), nondet w = withdrawals.oneOf() l1Relay(w) },
      fakeCaller(route, m, d, t),
      all { deposits != Set(), nondet dep = deposits.oneOf() expire(dep) },
      refund(m),
    }
  }

  // ---- Properties ----
  // ETH is never both delivered on the destination and refunded on the source.
  val NoDoubleSpend = MSGS.forall(m => not(relayed.get(m) and refunded.get(m)))
  val RefundImpliesExpired = MSGS.forall(m => refunded.get(m) implies expired.get(m))
  // An expired message was never relayed, and no initiating event is within the window at the
  // destination's clock (which only moves forward, so it never will be again).
  val ExpiredImpliesNeverRelayable = MSGS.forall(m =>
    expired.get(m) implies (not(relayed.get(m)) and not(withinWindow(events.get(m), now.get(DST.get(m)))))
  )
  // Every fact that expired a message was exported by that message's destination.
  val NoForgedFact = not(forgedAccepted)
  val AtMostOneRefund = MSGS.forall(m => refunds.get(m) <= 1)
  // Defense in depth (the messenger's target rule): once upgraded, the L2ToL2CrossDomainMessenger
  // never initiates a withdrawal. Safety does NOT depend on it (see safeNoTargetRule).
  val MessengerSilentAfterUpgrade = not(messengerSpokeAfterUpgrade)
  val Safety = NoDoubleSpend and RefundImpliesExpired and ExpiredImpliesNeverRelayable and NoForgedFact
    and AtMostOneRefund

  // ---- Non-vacuity witnesses (each must be VIOLATED in the safe instances) ----
  val NoRefundEver = MSGS.forall(m => not(refunded.get(m)))
  // A relay exactly at the window edge (exec - init == W) happens.
  val NoEdgeRelay = not(edgeRelay)
  // m2 (source B, destination A) is refunded: the fact is exported on A and routed to B.
  val NoRefundOfM2 = not(refunded.get("m2"))
  // m3 is refunded: its destination C joined the lockbox after genesis.
  val NoRefundOfM3 = not(refunded.get("m3"))
}

// ---------------------------------------------------------------------------------------------
// Instances. W and P in days. "safe" is the design: the exporter is trusted, all three L1 checks,
// lockbox joins only for standard chains, W = 7, P = 8.
// ---------------------------------------------------------------------------------------------
module safe {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// The target rule is defense in depth in this design: without it, the trusted sender still cannot
// be made to speak.
module safeNoTargetRule {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = false, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// No margin: P == W is still safe with the strict check.
module safeNoMargin {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 7, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// A shorter dependency-set window only delays expiry.
module safeShorterWindow {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 5,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// A resend that restarts the timestamp and refuses expired messages would also be safe.
module safeResendRestarts {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = true, RESEND_RESTARTS = true, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// ---- Each must produce a NoDoubleSpend counterexample ----

// The earlier design (trust the messenger): withdrawals from 0x..23 made before the target rule
// existed are forged facts.
module messengerTrustedPrestaged {
  import expiry(TRUSTED_SENDER = "messenger", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// Post-upgrade path only: pre-upgrade forgeries are switched off to isolate it.
module messengerTrustedNoTargetRule {
  import expiry(TRUSTED_SENDER = "messenger", TARGET_RULE = false, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = false).*
}

// P_contract < W_protocol (cap-compatible: W = 7, P = 6).
module periodBelowWindow {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 6, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// `>=` instead of `>` with no margin.
module expireGeNoMargin {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 7, EXPIRE_GE = true, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

module noRealMessengerCheck {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = false,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

module noLockboxCheck {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = false, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

module noSenderCheck {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = false, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// A chain that ran arbitrary code joins the lockbox later; its old withdrawals become trusted.
module nonstandardJoin {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = false, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// The original #22601 flaw: a resend that does not restart the timestamp.
module resendNoRestart {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = true, RESEND_RESTARTS = false, UNIQUE_IDS = true, PRE_UPGRADE_FORGERY = true).*
}

// Two lockbox members with the same L2 chain ID: C exports "not relayed" for B's message.
module duplicateChainId {
  import expiry(TRUSTED_SENDER = "exporter", TARGET_RULE = true, REAL_MESSENGER_CHECK = true,
    LOCKBOX_CHECK = true, SENDER_CHECK = true, JOIN_REQUIRES_STANDARD = true, PROTOCOL_WINDOW = 7,
    CONTRACT_PERIOD = 8, EXPIRE_GE = false, RESEND = false, RESEND_RESTARTS = false,
    UNIQUE_IDS = false, PRE_UPGRADE_FORGERY = true).*
}
