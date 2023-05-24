package store

import "encoding/json"

func serializeScoresV0(scores scoreRecord) ([]byte, error) {
	// v0 just serializes to JSON. New/unrecognized values default to 0.
	return json.Marshal(&scores)
}

func deserializeScoresV0(data []byte) (scoreRecord, error) {
	var out scoreRecord
	err := json.Unmarshal(data, &out)
	return out, err
}
