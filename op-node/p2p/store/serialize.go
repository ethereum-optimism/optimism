package store

import (
	"bytes"
	"encoding/binary"
	"time"
)

func serializeScoresV0(scores scoreRecord) ([]byte, error) {
	var b bytes.Buffer
	err := binary.Write(&b, binary.BigEndian, scores.lastUpdate.UnixMilli())
	if err != nil {
		return nil, err
	}
	err = binary.Write(&b, binary.BigEndian, scores.Gossip)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func deserializeScoresV0(data []byte) (scoreRecord, error) {
	var scores scoreRecord
	r := bytes.NewReader(data)
	var lastUpdate int64
	err := binary.Read(r, binary.BigEndian, &lastUpdate)
	if err != nil {
		return scoreRecord{}, err
	}
	scores.lastUpdate = time.UnixMilli(lastUpdate)
	err = binary.Read(r, binary.BigEndian, &scores.Gossip)
	if err != nil {
		return scoreRecord{}, err
	}
	return scores, nil
}
