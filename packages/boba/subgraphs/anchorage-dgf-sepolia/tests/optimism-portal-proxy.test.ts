import {
  assert,
  describe,
  test,
  clearStore,
  beforeAll,
  afterAll
} from "matchstick-as"
import { Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import { DisputeGameBlacklisted } from "../generated/schema"
import { DisputeGameBlacklisted as DisputeGameBlacklistedEvent } from "../generated/OptimismPortal2/OptimismPortal2"
import { handleDisputeGameBlacklisted } from "../src/optimism-portal-2"
import { createDisputeGameBlacklistedEvent } from "./optimism-portal-2-utils"

// Tests structure (matchstick-as >=0.5.0)
// https://thegraph.com/docs/en/developer/matchstick/#tests-structure-0-5-0

describe("Describe entity assertions", () => {
  beforeAll(() => {
    let disputeGame = Address.fromString(
      "0x0000000000000000000000000000000000000001"
    )
    let newDisputeGameBlacklistedEvent =
      createDisputeGameBlacklistedEvent(disputeGame)
    handleDisputeGameBlacklisted(newDisputeGameBlacklistedEvent)
  })

  afterAll(() => {
    clearStore()
  })

  // For more test scenarios, see:
  // https://thegraph.com/docs/en/developer/matchstick/#write-a-unit-test

  test("DisputeGameBlacklisted created and stored", () => {
    assert.entityCount("DisputeGameBlacklisted", 1)

    // 0xa16081f360e3847006db660bae1c6d1b2e17ec2a is the default address used in newMockEvent() function
    assert.fieldEquals(
      "DisputeGameBlacklisted",
      "0xa16081f360e3847006db660bae1c6d1b2e17ec2a-1",
      "disputeGame",
      "0x0000000000000000000000000000000000000001"
    )

    // More assert options:
    // https://thegraph.com/docs/en/developer/matchstick/#asserts
  })
})
