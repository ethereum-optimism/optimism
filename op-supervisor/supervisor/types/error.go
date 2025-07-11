package types

import "errors"

var (
	// ErrOutOfOrder happens when you try to add data to the DB,
	// but it does not actually fit onto the latest data (by being too old or new).
	ErrOutOfOrder = errors.New("data out of order")
	// ErrDataCorruption happens when the underlying DB has some I/O issue
	ErrDataCorruption = errors.New("data corruption")
	// ErrNotExact happens when we search the DB, know the data may be there, but is not (e.g. different revision)
	ErrNotExact = errors.New("missed data")
	// ErrSkipped happens when we try to retrieve data that is not available (pruned)
	// It may also happen if we erroneously skip data, that was not considered a conflict, if the DB is corrupted.
	ErrSkipped = errors.New("skipped data")
	// ErrFuture happens when data is just not yet available
	ErrFuture = errors.New("future data")
	// ErrInvalidatedRead happens when something was assumed from the DB, but then invalidated due to e.g. a reorg.
	ErrInvalidatedRead = errors.New("invalidated read")
	// ErrAlreadyInvalidatingRead happens when something is being invalidated,
	// and something else attempts to invalidate at the same time.
	ErrAlreadyInvalidatingRead = errors.New("already invalidating read")
	// ErrRewindFailed happens when we fail to rewind the chain (reorg response).
	ErrRewindFailed = errors.New("rewind failed")
	// ErrIneffective happens when data is accepted as compatible, but did not change anything.
	// This happens when a node is deriving an L2 block we already know of being derived from the given source,
	// but without path to skip forward to newer source blocks without doing the known derivation work first.
	ErrIneffective = errors.New("ineffective data")
	// ErrConflict happens when we know for sure that there is different canonical data
	ErrConflict = errors.New("conflicting data")
	// ErrAwaitReplacementBlock happens when we know for sure that a replacement block is needed before progress can be made.
	ErrAwaitReplacementBlock = errors.New("awaiting replacement block")
	// ErrStop can be used in iterators to indicate iteration has to stop
	ErrStop = errors.New("iter stop")
	// ErrOutOfScope is when data is accessed, but access is not allowed, because of a limited scope.
	// E.g. when limiting scope to L2 blocks derived from a specific subset of the L1 chain.
	ErrOutOfScope = errors.New("out of scope")
	// ErrPreviousToFirst is when you try to get the previous block of the first block
	// E.g. when calling PreviousDerivedFrom on the first L1 block in the DB.
	ErrPreviousToFirst = errors.New("cannot get parent of first block in the database")
	// ErrUnknownChain is when a chain is unknown, not in the dependency set.
	ErrUnknownChain = errors.New("unknown chain")
	// ErrNoRPCSource happens when a sub-service needs an RPC data source, but is not configured with one.
	ErrNoRPCSource = errors.New("no RPC client configured")
	// ErrUninitialized happens when a chain database is not initialized yet
	ErrUninitialized = errors.New("uninitialized chain database")
	// ErrFailsafeEnabled is when failsafe is enabled and the request is rejected
	ErrFailsafeEnabled = errors.New("failsafe is enabled, rejecting all CheckAccessList requests")
)

var genericInvalidParamsErr = -32602

// errorCodeMap is based on the interop supervisor spec - https://github.com/ethereum-optimism/specs/blob/28a0fac2428b10f9ee29ee1bfbbe366181cc9ac4/specs/interop/supervisor.md#json-rpc-error-codes
var errorCodeMap = map[error]int{
	ErrOutOfOrder:            -320900,
	ErrDataCorruption:        -321501,
	ErrNotExact:              -321500,
	ErrSkipped:               -320500,
	ErrFuture:                -321401,
	ErrIneffective:           -320601,
	ErrConflict:              -320600,
	ErrAwaitReplacementBlock: -320901,
	ErrStop:                  -321000,
	ErrOutOfScope:            -321100,
	ErrPreviousToFirst:       -321200,
	ErrUnknownChain:          -320501,
	ErrUninitialized:         -320400,

	ErrNoRPCSource:             genericInvalidParamsErr,
	ErrFailsafeEnabled:         genericInvalidParamsErr,
	ErrInvalidatedRead:         genericInvalidParamsErr,
	ErrAlreadyInvalidatingRead: genericInvalidParamsErr,
	ErrRewindFailed:            genericInvalidParamsErr,
}

// GetErrorCode returns the error code for the given error based on interop supervisor spec
func GetErrorCode(err error) int {
	if err == nil {
		return 0
	}
	for knownErr, code := range errorCodeMap {
		if errors.Is(err, knownErr) {
			return code
		}
	}
	return genericInvalidParamsErr
}
