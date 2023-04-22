package challenger

// Start begins the Challenger's engine driver,
// adding an instance to the waitgroup.
func (c *Challenger) Start() error {
	c.wg.Add(1)
	go c.drive()
	return nil
}

// Stop shuts down the Challenger.
func (c *Challenger) Stop() {
	c.cancel()
	close(c.done)
	c.wg.Wait()
}

// drive executes the Challenger's drive train.
func (c *Challenger) drive() {
	defer c.wg.Done()

	// Spawn the Oracle Hook
	c.wg.Add(1)
	go c.oracle()

	// Spawn the Factory Hook
	c.wg.Add(1)
	go c.factory()
}
