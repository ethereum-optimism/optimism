// -*- mode: Bluespec; -*-
// Unsafe instances of the rollout model (rollout.qnt), part 2: each drops one activation
// condition and must produce a NoDoubleSpend counterexample. They live in files of their own, in
// two parts, so that each file's flattened model stays within the Apalache server's input limit.

// AC3 dropped (members): a lockbox member's governance installs non-standard exporter code.
module rogueExporterMember {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = true,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val FR = fact("m1", "B", 10, "A", "B", false)
  run cex = init.then(nutTip("A")).then(l1Set("A", "new")).then(setExporter("B", "rogue"))
    .then(send("m1", 1)).then(exportRogue("B", "m1", "B", 10, "A"))
    .then(l1Relay(wd("B", "exporter", FR, true))).then(expire(dep("A", FR))).then(refund("m1"))
    .then(relay("m1", 7, 1)).expect(not(NoDoubleSpend))
}

// AC3 dropped (joiners): C ran non-standard exporter code while outside, then joins.
module rogueThenJoin {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = false, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val FR = fact("m1", "B", 10, "A", "C", false)
  run cex = init.then(setExporter("C", "rogue")).then(exportRogue("C", "m1", "B", 10, "A"))
    .then(setExporter("C", "std")).then(join("C")).then(nutTip("A")).then(l1Set("A", "new"))
    .then(send("m1", 1)).then(l1Relay(wd("C", "exporter", FR, false))).then(expire(dep("A", FR)))
    .then(refund("m1")).then(relay("m1", 7, 1)).expect(not(NoDoubleSpend))
}

// AC4 dropped: C has B's L2 chain ID and joins.
module duplicateChainId {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = false, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val FD = fact("m1", "B", 10, "A", "C", true)
  // Apalache's shortest counterexample (9 steps) needs no join: B (a member) exports for m3, whose
  // destination C shares B's chain ID, and C, which is not a member, relays it.
  pure val FN = fact("m3", "B", 10, "A", "B", true)
  run cexNonMember = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new"))
    .then(send("m3", 1)).then(exportStd("B", "m3", "A", 10))
    .then(l1Relay(wd("B", "exporter", FN, true))).then(expire(dep("A", FN))).then(refund("m3"))
    .then(relay("m3", 7, 1)).expect(not(NoDoubleSpend) and not(cluster.contains("C")))
  run cex = init.then(nutTip("A")).then(nutTip("C")).then(l1Set("A", "new")).then(join("C"))
    .then(send("m1", 1)).then(exportStd("C", "m1", "A", 10))
    .then(l1Relay(wd("C", "exporter", FD, true))).then(expire(dep("A", FD))).then(refund("m1"))
    .then(relay("m1", 7, 1)).expect(not(NoDoubleSpend))
}

// AC5 dropped: an L1CrossDomainMessenger of the earlier design (trusting 0x..23) is installed on L1
// at some point; 1.3.1 messengers have no target rule, so 0x..23 can be made to speak.
module l1EarlierDesign {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = true,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val FM = fact("m1", "B", 10, "A", "B", false)
  run cex = init.then(messengerForge("B", "m1", "B", 10, "A")).then(nutTip("A")).then(l1Set("A", "earlier"))
    .then(send("m1", 1)).then(l1Relay(wd("B", "messenger", FM, false))).then(expire(dep("A", FM)))
    .then(refund("m1")).then(relay("m1", 7, 1)).expect(not(NoDoubleSpend))
}

// The devstack's window check dropped: A launched with period 5 < W = 7. B exports at its time 7
// > 1 + 5, A expires and refunds, and B still relays at 8 <= 1 + 7.
module periodOverrideUnguarded {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "unguarded").* from "./rollout"

  pure val F7 = fact("m1", "B", 7, "A", "B", true)
  run cex = init.then(genesisWithPeriod("A", 5)).then(nutTip("B")).then(l1Set("A", "new"))
    .then(send("m1", 1)).then(exportStd("B", "m1", "A", 7))
    .then(l1Relay(wd("B", "exporter", F7, true))).then(expire(dep("A", F7))).then(refund("m1"))
    .then(relay("m1", 7, 8)).expect(not(NoDoubleSpend))
}
