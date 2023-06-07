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
				// c.ValidateOutput(log)
			case log := <-c.dgfLogs:
				c.log.Info("Dispatch received factory log", "log", log)
			}
		}
	}()
}
