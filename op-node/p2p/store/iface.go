package store

import (
	"errors"
	"math"
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type TopicScores struct {
	TimeInMesh               float64 `json:"timeInMesh"` // in seconds
	FirstMessageDeliveries   float64 `json:"firstMessageDeliveries"`
	MeshMessageDeliveries    float64 `json:"meshMessageDeliveries"`
	InvalidMessageDeliveries float64 `json:"invalidMessageDeliveries"`
}

type GossipScores struct {
	Total              float64     `json:"total"`
	Blocks             TopicScores `json:"blocks"` // fully zeroed if the peer has not been in the mesh on the topic
	IPColocationFactor float64     `json:"IPColocationFactor"`
	BehavioralPenalty  float64     `json:"behavioralPenalty"`
}

func (g GossipScores) Apply(rec *scoreRecord) {
	rec.PeerScores.Gossip = g
}

type ReqRespScores struct {
	ValidResponses   float64 `json:"validResponses"`
	ErrorResponses   float64 `json:"errorResponses"`
	RejectedPayloads float64 `json:"rejectedPayloads"`
}

type IncrementValidResponses struct {
	Cap float64
}

func (i IncrementValidResponses) Apply(rec *scoreRecord) {
	rec.PeerScores.ReqResp.ValidResponses = math.Min(rec.PeerScores.ReqResp.ValidResponses+1, i.Cap)
}

type IncrementErrorResponses struct {
	Cap float64
}

func (i IncrementErrorResponses) Apply(rec *scoreRecord) {
	rec.PeerScores.ReqResp.ErrorResponses = math.Min(rec.PeerScores.ReqResp.ErrorResponses+1, i.Cap)
}

type IncrementRejectedPayloads struct {
	Cap float64
}

func (i IncrementRejectedPayloads) Apply(rec *scoreRecord) {
	rec.PeerScores.ReqResp.RejectedPayloads = math.Min(rec.PeerScores.ReqResp.RejectedPayloads+1, i.Cap)
}

type DecayApplicationScores struct {
	ValidResponseDecay   float64
	ErrorResponseDecay   float64
	RejectedPayloadDecay float64
	DecayToZero          float64
}

func (d *DecayApplicationScores) Apply(rec *scoreRecord) {
	decay := func(value float64, decay float64) float64 {
		value *= decay
		if value < d.DecayToZero {
			return 0
		}
		return value
	}
	rec.PeerScores.ReqResp.ValidResponses = decay(rec.PeerScores.ReqResp.ValidResponses, d.ValidResponseDecay)
	rec.PeerScores.ReqResp.ErrorResponses = decay(rec.PeerScores.ReqResp.ErrorResponses, d.ErrorResponseDecay)
	rec.PeerScores.ReqResp.RejectedPayloads = decay(rec.PeerScores.ReqResp.RejectedPayloads, d.RejectedPayloadDecay)
}

type PeerScores struct {
	Gossip  GossipScores  `json:"gossip"`
	ReqResp ReqRespScores `json:"reqResp"`
}

// ScoreDatastore defines a type-safe API for getting and setting libp2p peer score information
type ScoreDatastore interface {
	// GetPeerScores returns the current scores for the specified peer
	GetPeerScores(id peer.ID) (PeerScores, error)

	// GetPeerScore returns the current combined score for the specified peer
	GetPeerScore(id peer.ID) (float64, error)

	// SetScore applies the given store diff to the specified peer
	SetScore(id peer.ID, diff ScoreDiff) (PeerScores, error)
}

// ScoreDiff defines a type-safe batch of changes to apply to the peer-scoring record of the peer.
// The scoreRecord the diff is applied to is private: diffs can only be defined in this package,
// to ensure changes to the record are non-breaking.
type ScoreDiff interface {
	Apply(score *scoreRecord)
}

var ErrUnknownBan = errors.New("unknown ban")

type PeerBanStore interface {
	// SetPeerBanExpiration create the peer ban with expiration time.
	// If expiry == time.Time{} then the ban is deleted.
	SetPeerBanExpiration(id peer.ID, expiry time.Time) error
	// GetPeerBanExpiration gets the peer ban expiration time, or ErrUnknownBan error if none exists.
	GetPeerBanExpiration(id peer.ID) (time.Time, error)
}

type IPBanStore interface {
	// SetIPBanExpiration create the IP ban with expiration time.
	// If expiry == time.Time{} then the ban is deleted.
	SetIPBanExpiration(ip net.IP, expiry time.Time) error
	// GetIPBanExpiration gets the IP ban expiration time, or ErrUnknownBan error if none exists.
	GetIPBanExpiration(ip net.IP) (time.Time, error)
}

type MetadataStore interface {
	// SetPeerMetadata sets the metadata for the specified peer
	SetPeerMetadata(id peer.ID, md PeerMetadata) (PeerMetadata, error)
	// GetPeerMetadata returns the metadata for the specified peer
	GetPeerMetadata(id peer.ID) (PeerMetadata, error)
}

// ExtendedPeerstore defines a type-safe API to work with additional peer metadata based on a libp2p peerstore.Peerstore
type ExtendedPeerstore interface {
	peerstore.Peerstore
	ScoreDatastore
	peerstore.CertifiedAddrBook
	PeerBanStore
	IPBanStore
	MetadataStore
}
