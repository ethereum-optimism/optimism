package render

import "github.com/ethereum-optimism/optimism/op-private-interop/wire"

// These aliases preserve the rendering API; wire owns the call encodings used by derivation.
type SentMessage = wire.SentMessage

var (
	replayEventArgs           = wire.ReplayEventArgs
	SentMessageEventTopic     = wire.SentMessageEventTopic
	RelayedMessageEventTopic  = wire.RelayedMessageEventTopic
	ReplaySentMessageSig      = wire.ReplaySentMessageSig
	ReplayEventSig            = wire.ReplayEventSig
	PostClaimSig              = wire.PostClaimSig
	ReplaySentMessageSelector = wire.ReplaySentMessageSelector
	ReplayEventSelector       = wire.ReplayEventSelector
	PostClaimSelector         = wire.PostClaimSelector
	EncodePostClaim           = wire.EncodePostClaim
	DecodeSentMessage         = wire.DecodeSentMessage
	EncodeReplaySentMessage   = wire.EncodeReplaySentMessage
	EncodeReplayEvent         = wire.EncodeReplayEvent
	replaySentMessageArgs     = wire.ReplaySentMessageArgs
)
