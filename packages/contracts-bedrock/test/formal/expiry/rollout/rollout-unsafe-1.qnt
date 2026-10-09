// -*- mode: Bluespec; -*-
// Unsafe instances of the rollout model (rollout.qnt), part 1: each drops one activation
// condition and must produce a NoDoubleSpend counterexample. They live in files of their own, in
// two parts, so that each file's flattened model stays within the Apalache server's input limit.

// AC1 dropped: B's exporter is live while B relays without a W rule.
module noWindowRuleOnB {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(7, 99), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  run cex = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new")).then(honestTail1)
    .then(relay("m1", 99, 10)).expect(not(NoDoubleSpend))
}

// AC1 dropped on one chain only: C (which joins mid-rollout) uses W = 9 > P = 8 at times.
module windowAbovePOnC {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(7, 9)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val F3 = fact("m3", "C", 10, "A", "C", true)
  run cex = init.then(nutTip("A")).then(nutTip("C")).then(l1Set("A", "new")).then(join("C"))
    .then(send("m3", 1)).then(exportStd("C", "m3", "A", 10))
    .then(l1Relay(wd("C", "exporter", F3, true))).then(expire(dep("A", F3))).then(refund("m3"))
    .then(relay("m3", 9, 10)).expect(not(NoDoubleSpend))
}

// AC2 dropped: governance rolls a source's messenger back to 1.3.1 (resendMessage returns).
module govMessengerDowngrade {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "any", L2CM_DOWNGRADE_GUARD = true, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  run cex = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new")).then(honestTail1)
    .then(govMessengerDowngrade("A")).then(resend("m1", 12)).then(relay("m1", 7, 15))
    .expect(not(NoDoubleSpend))
}

// AC2's code half dropped: without the L2ContractsManager's semver guard, the locked Lagoon bundle
// activating after the tip bundle downgrades the messenger.
module lagoonWithoutGuard {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", L2CM_DOWNGRADE_GUARD = false, ROGUE_EXPORTER_MEMBER = false,
    ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true, L1_EARLIER_DESIGN = false,
    UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  run cex = init.then(nutTip("A")).then(nutTip("B")).then(l1Set("A", "new")).then(honestTail1)
    .then(nutLagoon("A")).then(resend("m1", 12)).then(relay("m1", 7, 15))
    .expect(not(NoDoubleSpend))
}
