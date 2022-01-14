package node

import "context"

type OpNodeCmd struct {
}

func (c *OpNodeCmd) Help() string {
	return "Run optimism node"
}

func (c *OpNodeCmd) Run(ctx context.Context, args ...string) error {
	return nil
}

func (c *OpNodeCmd) Close() error {
	return nil
}
