package main

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

type mockFormat struct {
	output aggregate
}

func (m *mockFormat) readAggregate(file string) (aggregate, error) {
	first, err := strconv.ParseInt(file, 10, 64)
	if err != nil {
		return aggregate{}, err
	}
	return aggregate{
		ChainID: 1,
		First:   uint64(first),
		Last:    uint64(first),
		Results: map[uint64][]uint64{
			1: {1, 2, 3},
			2: {4, 5, 6},
		},
	}, nil
}

func (m *mockFormat) writeAggregate(a aggregate, output string) error {
	m.output = a
	return nil
}

func TestMerge(t *testing.T) {
	m := &mockFormat{}
	formats["json"] = m

	app := cli.NewApp()
	app.Commands = []*cli.Command{mergeCommand}
	ctx := cli.NewContext(app, nil, nil)
	err := mergeCommand.Run(ctx, "merge", "--files", "1,2,3")
	assert.NoError(t, err)

	assert.Equal(t, aggregate{
		ChainID: 1,
		First:   1,
		Last:    3,
		Results: map[uint64][]uint64{
			1: {1, 2, 3},
			2: {4, 5, 6},
		},
	}, m.output)
}
