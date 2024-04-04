package op_service

// This file pins dependencies that have broken releases.
//
// Pebble uses sentry-go
// sentry-go uses a deleted release of github.com/kataras/iris/v12
// And Go is then unable to resolve the sentry-dependency due to a missing indirect
//
// So we pin iris, to then explicitly define an actual present release.
//
// Also see https://github.com/ethereum/go-ethereum/issues/28036
// Once op-geth is updated with more recent upstream changes,
// the indirect dependencies are fixed, solving the iris dependency resolution issue.

import (
	_ "github.com/kataras/iris/v12"
)
