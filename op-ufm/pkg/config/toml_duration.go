package config

import "time"

type TOMLDuration time.Duration

func (t *TOMLDuration) UnmarshalText(b []byte) error {
	d, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}

	*t = TOMLDuration(d)
	return nil
}
