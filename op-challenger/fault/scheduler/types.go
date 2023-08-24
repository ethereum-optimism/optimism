package scheduler

import "context"

type GamePlayer interface {
	ProgressGame(ctx context.Context) bool
}

type job struct {
	player   GamePlayer
	resolved bool
}
