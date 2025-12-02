package bindings

type AnchorStateRegistry struct {
	SetRespectedGameType func(gameType uint32) TypedCall[any] `sol:"setRespectedGameType"`
}
