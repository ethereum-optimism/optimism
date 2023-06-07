package challenger

// dispatch runs the dispatch service for the challenger.
// This function spawns a goroutine.
func (c *Challenger) dispatch() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				c.log.Info("Dispatch quitting")
				return
			case log := <-c.l2ooLogs:
				c.log.Info("Dispatch received oracle log", "log", log)
				l2BlockNumber, expected, err := c.ParseOutputLog(&log)
				if err != nil {
					c.log.Error("Failed to parse output log", "err", err)
					continue
				}
				valid, output, err := c.ValidateOutput(c.ctx, l2BlockNumber, expected)
				if err != nil {
					c.log.Error("Failed to validate output", "err", err)
					continue
				}
				if !valid {
					c.log.Error("Output is invalid", "expected", expected, "output", output)
					// moose: here we should spawn a new goroutine to challenge the output
					// moose: we need to check:
					// moose: a) that we have not already submitted a output dispute
					// moose: b) if a game, or multiple games, exist for this output
					// moose: c) if we need to create a dispute game for the output
				}
			case log := <-c.dgfLogs:
				c.log.Info("Dispatch received factory log", "log", log)
			}
		}
	}()
}
