package challenger

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// indexer runs the indexing service for the challenger.
// This function spawns a goroutine to run the indexer.
func (c *Challenger) indexer() {
	c.l2ooLogs = make(chan types.Log)
	c.dgfLogs = make(chan types.Log)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		// Create an L2oo subscription
		oracleSub, err := c.NewOracleSubscription()
		if err != nil {
			c.log.Error("Failed to build oracle log filter", "err", err)
			return
		}
		err = oracleSub.Subscribe()
		if err != nil {
			c.log.Error("Failed to subscribe to oracle output proposals", "err", err)
			return
		}
		defer oracleSub.Quit()
		// moose: add a traversal client here

		// Create an dgf subscription
		factorySub, err := c.NewFactorySubscription()
		if err != nil {
			c.log.Error("Failed to build factory log filter", "err", err)
			return
		}
		err = factorySub.Subscribe()
		if err != nil {
			c.log.Error("Failed to subscribe to dispute game factory events", "err", err)
			return
		}
		defer factorySub.Quit()
		// moose: add a traversal client here

		// Listen to the subscription logs
		for {
			select {
			case <-c.ctx.Done():
				c.log.Info("Indexer quitting")
				return
			case log := <-oracleSub.Logs():
				c.log.Info("Indexer received oracle log", "log", log)
				c.l2ooLogs <- log
			case log := <-factorySub.Logs():
				c.log.Info("Indexer received factory log", "log", log)
				c.dgfLogs <- log
			}
		}
	}()
}
