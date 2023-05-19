package store

import (
	"bytes"
	"encoding/binary"

	pool "github.com/libp2p/go-buffer-pool"
)

func serializeScoresV0(scores PeerScores) ([]byte, error) {
	var b pool.Buffer
	err := binary.Write(&b, binary.BigEndian, scores.Gossip)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func deserializeScoresV0(data []byte) (PeerScores, error) {
	var scores PeerScores
	r := bytes.NewReader(data)
	err := binary.Read(r, binary.BigEndian, &scores.Gossip)
	if err != nil {
		return PeerScores{}, err
	}
	return scores, nil
}
