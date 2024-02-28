// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	"math"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
)

const (
	NanosInSecond  = 1e9
	MillisInSecond = 1e3
)

// Moment is a wrapper around milliseconds/timestamps using the `time.Time` type.
type Moment struct {
	time.Time
}

// NewMoment creates a new Moment type
func NewMoment(t time.Time) Moment {
	return Moment{t}
}

func (m *Moment) Decode(decoder scale.Decoder) error {
	var u uint64
	err := decoder.Decode(&u)
	if err != nil {
		return err
	}

	// Error in case of overflow
	if u > math.MaxInt64 {
		return fmt.Errorf("cannot decode a uint64 into a Moment if it overflows int64")
	}

	secs := u / MillisInSecond
	nanos := (u % uint64(MillisInSecond)) * uint64(NanosInSecond)

	*m = NewMoment(time.Unix(int64(secs), int64(nanos)))

	return nil
}

func (m Moment) Encode(encoder scale.Encoder) error {
	err := encoder.Encode(uint64(m.UnixNano() / (NanosInSecond / MillisInSecond)))
	if err != nil {
		return err
	}

	return nil
}
