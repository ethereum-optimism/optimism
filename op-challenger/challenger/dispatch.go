package challenger

// dispatch runs the dispatch service for the challenger.
// This function spawns a goroutine.
func (c *Challenger) dispatch() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.dispatchSwitch()
	}()
}

// dispatchSwitch is the main loop for the dispatch service.
func (c *Challenger) dispatchSwitch() {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("Dispatch quitting")
			return
		case log := <-c.l2ooLogs:
			c.log.Info("Dispatch received oracle log", "log", log)
			proposal, err := c.ParseOutputLog(&log)
			if err != nil {
				c.log.Error("Failed to parse output log", "err", err)
				continue
			}
			valid, output, err := c.ValidateOutput(c.ctx, *proposal)
			if err != nil {
				c.log.Error("Failed to validate output", "err", err)
				continue
			}
			if !valid {
				c.log.Error("Output is invalid", "expected", proposal.OutputRoot, "output", output)
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
}
