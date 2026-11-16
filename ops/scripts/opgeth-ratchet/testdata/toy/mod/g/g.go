package g

import "example.com/geth/types"

func Gas() int { return types.Gas(21000) }

// Fails against the fork too, so it is no fork-API use the classifier can describe.
var Broken int = "broken"

type meters uint64

type feet uint64

var Mixed = meters(1) + feet(2)
