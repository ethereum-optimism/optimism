// -*- mode: Bluespec; -*-
// Unsafe instances of the rollout model (rollout.qnt), part 3: each drops one activation
// condition and must produce a NoDoubleSpend counterexample. They live in files of their own, in
// parts, so that each file's flattened model stays within the Apalache server's input limit.

// AC7 dropped: governance upgrades a 1.3.1 messenger, never initialized for expiry, to 2.0.0 with
// ProxyAdmin.upgrade alone. The stored period stays 0, so expireMessage accepts any fact dated after
// the send. B exports "not relayed at 2" for m1 (sent at 1), A expires and refunds it, and B relays
// m1 at 3, well within the window.
module govDirectUpgradeUninitialized {
  import rollout(CONTRACT_PERIOD = 8,
    WIN_BEFORE = Map("A" -> Set(7, 99), "B" -> Set(7, 99), "C" -> Set(7, 99)),
    WIN_AFTER = Map("A" -> Set(5, 7), "B" -> Set(5, 7), "C" -> Set(5, 7)),
    GOV_MSGR_DOWNGRADE = "beforeTimestamps", GOV_DIRECT_UPGRADE = "any", L2CM_DOWNGRADE_GUARD = true,
    ROGUE_EXPORTER_MEMBER = false, ROGUE_EXPORTER_OUTSIDER = true, JOIN_REQUIRES_CLEAN_EXPORTER = true,
    L1_EARLIER_DESIGN = false, UNIQUE_IDS = true, NEW_WRITES_SENT_MESSAGES = true,
    PERIOD_OVERRIDE = "guarded").* from "./rollout"

  pure val F2 = fact("m1", "B", 2, "A", "B", true)
  run cex = init.then(govMessengerUpgradeDirect("A"))
    .expect(period.get("A") == 0 and chains.get("A").msgr == "new")
    .then(setBridge("A", true)).then(nutTip("B")).then(l1Set("A", "new"))
    .then(send("m1", 1)).then(exportStd("B", "m1", "A", 2))
    .then(l1Relay(wd("B", "exporter", F2, true))).then(expire(dep("A", F2))).then(refund("m1"))
    .then(relay("m1", 7, 3)).expect(not(NoDoubleSpend))
}
